import grpc from 'k6/net/grpc';
import { sleep, check, fail } from 'k6';
import { scenario, vu } from 'k6/execution';
import { SharedArray } from 'k6/data';

const ADDR = __ENV.TARGET_ADDR || 'MISSING_ADDR';
const TEST_DURATION_SECONDS = __ENV.TEST_DURATION_SECONDS ? parseInt(__ENV.TEST_DURATION_SECONDS, 10) : 600;
const WATCH_GRACE_PERIOD_SECONDS = __ENV.WATCH_GRACE_PERIOD_SECONDS ? parseInt(__ENV.WATCH_GRACE_PERIOD_SECONDS, 10) : 30;
const GPU_COUNT = __ENV.GPU_COUNT ? parseInt(__ENV.GPU_COUNT, 10) : 512;

const TEST_DURATION = `${TEST_DURATION_SECONDS}s`;
const WATCH_GRACE_PERIOD = `${WATCH_GRACE_PERIOD_SECONDS}s`;
const GPU_NAMES = new SharedArray('gpu names', function () {
    return Array.from({ length: GPU_COUNT }, (_, i) => `gpu-${i}`);
});

const HEARTBEAT_END = Math.floor(GPU_NAMES.length * 0.5);
const WATCHDOG_END = Math.floor(GPU_NAMES.length * 0.75);

const EXPECTED_CREATION_EVENTS = GPU_COUNT; 
const EXPECTED_HEARTBEAT_EVENTS = (TEST_DURATION_SECONDS / 10) * HEARTBEAT_END;
const EXPECTED_WATCHDOG_EVENTS = (TEST_DURATION_SECONDS / 30) * (WATCHDOG_END - HEARTBEAT_END);
const EXPECTED_EXTERNAL_MONITOR_EVENTS = (TEST_DURATION_SECONDS / 30) * (GPU_COUNT - WATCHDOG_END);
const TOTAL_EXPECTED_EVENTS = EXPECTED_CREATION_EVENTS + EXPECTED_HEARTBEAT_EVENTS + EXPECTED_WATCHDOG_EVENTS + EXPECTED_EXTERNAL_MONITOR_EVENTS;

export const options = {
    summaryTrendStats: ['med', 'p(95)', 'p(99)', 'max'],
    scenarios: {
        heartbeat: {
            executor: 'constant-arrival-rate',
            rate: GPU_COUNT,
            timeUnit: '30s',
            duration: TEST_DURATION,
            preAllocatedVUs: Math.max(20, Math.floor(GPU_COUNT / 10)),
            exec: 'heartbeat',
        },
        internal_watchdog: {
            executor: 'constant-arrival-rate',
            rate: (WATCHDOG_END - HEARTBEAT_END),
            timeUnit: '30s',
            duration: TEST_DURATION,
            preAllocatedVUs: 20,
            exec: 'internalWatchdog',
        },
        external_monitor: {
            executor: 'constant-arrival-rate',
            rate: (GPU_NAMES.length - WATCHDOG_END),
            timeUnit: '30s',
            duration: TEST_DURATION,
            preAllocatedVUs: 20,
            exec: 'externalMonitor',
        },
        reconciler: {
            executor: 'constant-arrival-rate',
            rate: Math.floor(GPU_COUNT / 2),
            timeUnit: '10s',
            duration: TEST_DURATION,
            preAllocatedVUs: 10,
            exec: 'reconcile',
        },
        watcher: {
            executor: 'per-vu-iterations',
            vus: 3,
            iterations: 1,
            maxDuration: TEST_DURATION + WATCH_GRACE_PERIOD,
            exec: 'watch',
        },
        controller_resync: {
          executor: 'constant-arrival-rate',
            rate: 1,
            timeUnit: '1m',
            duration: TEST_DURATION,
            preAllocatedVUs: 1,
            exec: 'list',
        },
    },
    thresholds: {
        'checks': ['rate>0.999'],

        'grpc_req_duration{rpc:create}': ['p(95)<15', 'p(99)<30'],
        'grpc_req_duration{rpc:get}': ['p(95)<15', 'p(99)<30'],
        'grpc_req_duration{rpc:update_status}': ['p(95)<15', 'p(99)<30'],
        'grpc_req_duration{rpc:list}': ['p(95)<20', 'p(99)<40'],
        'grpc_req_duration{rpc:delete}': ['p(95)<15', 'p(99)<30'],
    },
};


const client = new grpc.Client();
client.load(['/test'], 'gpu.proto');

function connect() {
    if (vu.iterationInScenario === 0) {
        if (!client.connect(ADDR, { plaintext: true })) {
            fail(`VU ${vu.idInTest} could not connect to ${ADDR}`);
        }
    }
}

export function setup() {
    if (ADDR === 'MISSING_ADDR') fail("FATAL: TARGET_ADDR environment variable is required.");
    if (!__ENV.TEST_DURATION_SECONDS) fail("FATAL: TEST_DURATION_SECONDS environment variable is required.");
    if (!__ENV.WATCH_GRACE_PERIOD_SECONDS) fail("FATAL: WATCH_GRACE_PERIOD_SECONDS environment variable is required.");
    if (!__ENV.GPU_COUNT) fail("FATAL: GPU_COUNT environment variable is required.");

    if (!client.connect(ADDR, { plaintext: true })) {
        fail(`FATAL: Setup could not connect to ${ADDR}`);
    }

    console.log(`Preparing ${GPU_NAMES.length} GPU environment...`);
    GPU_NAMES.forEach((name) => {
        const jitter = (Math.random() < 0.7) ? (Math.random() * 0.01) : (0.1 + Math.random() * 0.2);
        sleep(jitter);

        const payload = {
            gpu: {
                metadata: { name: name, namespace: 'default' },
                spec: { uuid: `uuid-${name}` },
                status: { conditions: [{ type: 'Ready', status: 'False', reason: 'Init' }] }
            }
        };
        const res = client.invoke('nvidia.nvsentinel.v1alpha1.GpuService/CreateGpu', payload, { tags: { rpc: 'create', caller: 'setup' } });
        check(res, { 'Create OK': (r) => r.status === grpc.StatusOK });
    });
    client.close();
    console.log(`Starting ${TEST_DURATION} test...`)
}

export function heartbeat() {
    connect();
    const name = GPU_NAMES[vu.idInTest % HEARTBEAT_END];
    updateStatus(name, 'heartbeat', 'Ready', 'True', 'DriverReady');
}

export function internalWatchdog() {
    connect();
    const poolSize = WATCHDOG_END - HEARTBEAT_END;
    const randomIndex = Math.floor(Math.random() * poolSize);
    const name = GPU_NAMES[randomIndex + HEARTBEAT_END];
    updateStatus(name, 'internal_watchdog', 'Ready', 'False', 'Unknown');
}

export function externalMonitor() {
    connect();
    const poolSize = GPU_NAMES.length - WATCHDOG_END;
    const randomIndex = Math.floor(Math.random() * poolSize);
    const name = GPU_NAMES[randomIndex + WATCHDOG_END];
    updateStatus(name, 'external_monitor', 'HardwareFailure', 'False', 'NoFailures');
}

export function reconcile() {
    sleep(Math.random() * 0.1);
    connect();
    const name = GPU_NAMES[vu.idInTest % GPU_NAMES.length];
    const res = client.invoke('nvidia.nvsentinel.v1alpha1.GpuService/GetGpu', { name, namespace: 'default' }, { tags: { rpc: 'get', caller: 'reconciler' } });
    check(res, { 'Get OK': (r) => r && r.status === grpc.StatusOK });
}

export function list() {
    sleep(Math.random() * 0.1);
    connect();
    const res = client.invoke('nvidia.nvsentinel.v1alpha1.GpuService/ListGpus', {}, { tags: { rpc: 'list', caller: 'controller' } });
    check(res, { 'List OK': (r) => r && r.status === grpc.StatusOK && r.message && r.message.gpuList && r.message.gpuList.items && r.message.gpuList.items.length === GPU_COUNT });
}

export function watch() {
    connect();

    const watchStartTime = Date.now();
    const endTime = watchStartTime + (TEST_DURATION_SECONDS * 1000);

    while (Date.now() < endTime) {
        let eventCount = 0;
        const stream = new grpc.Stream(client, 'nvidia.nvsentinel.v1alpha1.GpuService/WatchGpus', {});
        stream.on('data', () => { eventCount++; });
        stream.on('error', (err) => { console.error(`Stream error: ${err.message}`); });
        stream.on('end', () => { 
            if (eventCount === 0) {
                console.error(`✗ Watch OK '${eventCount}' events received`);
            }
        });
        stream.write({});

        const watchJitter = 40 + (Math.random() * 10); 
        sleep(watchJitter);

        stream.end()
        
        const pauseJitter = 1 + Math.random();
        sleep(pauseJitter);
    }
}

function updateStatus(name, caller, type, status, reason) {
    const now = new Date().getTime();
    const cycleSeconds = Math.floor(now / 1000) % 300;
    if (cycleSeconds >= 0 && cycleSeconds < 10) {
        return; 
    }

    const jitter = (Math.random() < 0.8) ? (Math.random() * 0.01) : (Math.random() * 0.05);
    sleep(jitter);

    const MAX_RETRIES = 3;
    let success = false;

    for (let i = 0; i < MAX_RETRIES; i++) {
      const getRes = client.invoke('nvidia.nvsentinel.v1alpha1.GpuService/GetGpu', {
          name: name, namespace: 'default'
      }, { tags: { rpc: 'get', caller: caller } });
      
      if (!getRes || getRes.status !== grpc.StatusOK) {
          check(getRes, { 'Get OK': () => false });
          return;
      }

      const payload = {
          gpu: {
              metadata: { 
                  name: name, namespace: 'default',
                  resourceVersion: String(getRes.message.gpu.metadata.resourceVersion)
              },
              status: {
                  conditions: [{ type: type, status: status, reason: reason, lastTransitionTime: new Date().toISOString() }]
              }
          }
      };

      const res = client.invoke('nvidia.nvsentinel.v1alpha1.GpuService/UpdateGpuStatus', payload, { 
          tags: { rpc: 'update_status', caller: caller } 
      });

      if (res.status === grpc.StatusOK) {
          check(res, { 'UpdateStatus OK': () => true });
          success = true;
          break; 
      } else if (res.status === 10) {
          console.warn(`[VU ${vu.idInTest}] Conflict on ${name} (retry ${i+1}/${MAX_RETRIES})`);
          sleep(Math.random() * 0.2 * (i+1));
          continue;
      } else {
          check(res, { 'UpdateStatus OK': () => false });
          break;
      }
    }
    
    if (!success) {
        fail(`failed to update GPU ${name} status (${caller})`);
    }
}

export function teardown() {
    console.log("Cleaning up test resources...");
    client.connect(ADDR, { plaintext: true });
    GPU_NAMES.forEach((name) => {
        const jitter = (Math.random() < 0.8) ? (Math.random() * 0.01) : (Math.random() * 0.05);
        sleep(jitter);

        const res = client.invoke('nvidia.nvsentinel.v1alpha1.GpuService/DeleteGpu', { name, namespace: 'default' }, {
          tags: { rpc: 'delete', caller: 'teardown' }
        });
        check(res, { 'Delete OK': (r) => r && r.status === grpc.StatusOK });
    });
    client.close();
}
