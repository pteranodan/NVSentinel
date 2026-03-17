/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Portions Copyright (c) 2026 NVIDIA CORPORATION. All rights reserved.

Modified from the original to support gRPC transport.
Origin: https://github.com/kubernetes/code-generator/blob/v0.34.1/cmd/client-gen/generators/generator_for_type.go
*/

// Package generators has the generators for the client-gen utility.
package generators

import (
	"io"
	"path"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"k8s.io/gengo/v2/generator"
	"k8s.io/gengo/v2/namer"
	"k8s.io/gengo/v2/types"

	"github.com/nvidia/nvsentinel/code-generator/cmd/client-gen/generators/util"
	clientgentypes "github.com/nvidia/nvsentinel/code-generator/cmd/client-gen/types"
)

// genClientForType produces a file for each top-level type.
type genClientForType struct {
	generator.GoGenerator
	outputPackage string // must be a Go import-path
	inputPackage  string
	group         string
	version       string
	groupGoName   string
	typeToMatch   *types.Type
	imports       namer.ImportTracker
	protoPackage  clientgentypes.ProtobufPackage
}

var _ generator.Generator = &genClientForType{}

var titler = cases.Title(language.Und)

// Filter ignores all but one type because we're making a single file per type.
func (g *genClientForType) Filter(c *generator.Context, t *types.Type) bool {
	return t == g.typeToMatch
}

func (g *genClientForType) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

func (g *genClientForType) Imports(c *generator.Context) (imports []string) {
	return append(
		g.imports.ImportLines(),
		g.protoPackage.ImportLines()...,
	)
}

// Ideally, we'd like genStatus to return true if there is a subresource path
// registered for "status" in the API server, but we do not have that
// information, so genStatus returns true if the type has a status field.
func genStatus(t *types.Type) bool {
	// Default to true if we have a Status member
	hasStatus := false
	for _, m := range t.Members {
		if m.Name == "Status" {
			hasStatus = true
			break
		}
	}
	return hasStatus && !util.MustParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...)).NoStatus
}

func (g *genClientForType) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "$", "$")
	pkg := path.Base(t.Name.Package)
	tags, err := util.ParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...))
	if err != nil {
		return err
	}
	protoType := titler.String(t.Name.Name)
	m := map[string]interface{}{
		"type":             t,
		"inputType":        t,
		"resultType":       t,
		"package":          pkg,
		"Package":          namer.IC(pkg),
		"namespaced":       !tags.NonNamespaced,
		"Group":            namer.IC(g.group),
		"GroupGoName":      g.groupGoName,
		"Version":          namer.IC(g.version),
		"ProtoType":        protoType,
		"context":          c.Universe.Type(types.Name{Package: "context", Name: "Context"}),
		"NewBadRequest":    c.Universe.Function(types.Name{Package: "k8s.io/apimachinery/pkg/api/errors", Name: "NewBadRequest"}),
		"GetOptions":       c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "GetOptions"}),
		"ListOptions":      c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "ListOptions"}),
		"CreateOptions":    c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "CreateOptions"}),
		"DeleteOptions":    c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "DeleteOptions"}),
		"UpdateOptions":    c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "UpdateOptions"}),
		"PatchOptions":     c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "PatchOptions"}),
		"PatchType":        c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/types", Name: "PatchType"}),
		"runtime":          c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/runtime", Name: "Object"}),
		"watchInterface":   c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/watch", Name: "Interface"}),
		"logr":             c.Universe.Type(types.Name{Package: "github.com/go-logr/logr", Name: "Logger"}),
		"nvgrpc":           c.Universe.Type(types.Name{Package: "github.com/nvidia/nvsentinel/pkg/grpc/client", Name: "NewWatcher"}),
		"NewAPIError":      c.Universe.Function(types.Name{Package: "github.com/nvidia/nvsentinel/pkg/grpc/errors", Name: "NewAPIError"}),
		"pb":               g.protoPackage.Alias,
		"apiPackage":       c.Universe.Type(types.Name{Package: g.inputPackage, Name: "Ignored"}),
		"Converter":        c.Universe.Type(types.Name{Package: g.inputPackage, Name: "Converter"}),
		"NewServiceClient": g.protoPackage.ServiceClientConstructorFor(protoType),
	}

	sw.Do(getterComment, m)
	if tags.NonNamespaced {
		sw.Do(getterNonNamespaced, m)
	} else {
		sw.Do(getterNamespaced, m)
	}

	sw.Do(generateInterfaceTemplate(tags, t), m)

	if tags.NonNamespaced {
		sw.Do(structTemplateNonNamespaced, m)
		sw.Do(constructorTemplateNonNamespaced, m)
	} else {
		sw.Do(structTemplateNamespaced, m)
		sw.Do(constructorTemplateNamespaced, m)
	}

	sw.Do(structHelpers, m)

	if tags.HasVerb("get") {
		sw.Do(getTemplate, m)
	}

	if tags.HasVerb("list") {
		sw.Do(listTemplate, m)
	}

	if tags.HasVerb("watch") {
		sw.Do(watchTemplate, m)
		sw.Do(watchAdapterTemplate, m)
	}

	if tags.HasVerb("create") {
		sw.Do(createTemplate, m)
	}

	if tags.HasVerb("update") {
		sw.Do(updateTemplate, m)
	}

	if genStatus(t) && tags.HasVerb("updateStatus") {
		sw.Do(updateStatusTemplate, m)
	}

	if tags.HasVerb("delete") {
		sw.Do(deleteTemplate, m)
	}

	if tags.HasVerb("patch") {
		sw.Do(patchTemplate, m)
	}

	return sw.Error()
}

func generateInterfaceTemplate(tags util.Tags, t *types.Type) string {
	lines := []string{}

	lines = append(lines, `
// $.type|public$Interface has methods to work with $.type|public$ resources.
type $.type|public$Interface interface {`)

	if tags.HasVerb("create") {
		lines = append(lines, `	Create(ctx $.context|raw$, $.type|allLowercase$ *$.type|raw$, opts $.CreateOptions|raw$) (*$.type|raw$, error)`)
	}
	if tags.HasVerb("update") {
		lines = append(lines, `	Update(ctx $.context|raw$, $.type|allLowercase$ *$.type|raw$, opts $.UpdateOptions|raw$) (*$.type|raw$, error)`)
	}
	if genStatus(t) && tags.HasVerb("updateStatus") {
		lines = append(lines, `	UpdateStatus(ctx $.context|raw$, $.type|allLowercase$ *$.type|raw$, opts $.UpdateOptions|raw$) (*$.type|raw$, error)`)
	}
	if tags.HasVerb("delete") {
		lines = append(lines, `	Delete(ctx $.context|raw$, name string, opts $.DeleteOptions|raw$) error`)
	}
	if tags.HasVerb("get") {
		lines = append(lines, `	Get(ctx $.context|raw$, name string, opts $.GetOptions|raw$) (*$.type|raw$, error)`)
	}
	if tags.HasVerb("list") {
		lines = append(lines, `	List(ctx $.context|raw$, opts $.ListOptions|raw$) (*$.type|raw$List, error)`)
	}
	if tags.HasVerb("watch") {
		lines = append(lines, `	Watch(ctx $.context|raw$, opts $.ListOptions|raw$) ($.watchInterface|raw$, error)`)
	}
	if tags.HasVerb("patch") {
		lines = append(lines, `	Patch(ctx $.context|raw$, name string, pt $.PatchType|raw$, data []byte, opts $.PatchOptions|raw$, subresources ...string) (*$.type|raw$, error)`)
	}

	lines = append(lines, `	$.type|public$Expansion
}
`)

	return strings.Join(lines, "\n")
}

var getterComment = `
// $.type|publicPlural$Getter has a method to return a $.type|public$Interface.
// A group's client should implement this interface.`

var getterNamespaced = `
type $.type|publicPlural$Getter interface {
	$.type|publicPlural$(namespace string) $.type|public$Interface
}
`

var getterNonNamespaced = `
type $.type|publicPlural$Getter interface {
	$.type|publicPlural$() $.type|public$Interface
}
`

var structTemplateNamespaced = `
// $.type|allLowercasePlural$ implements $.type|public$Interface
type $.type|allLowercasePlural$ struct {
	client    $.pb$.$.ProtoType$ServiceClient
	$.Converter|raw$
	logger    $.logr|raw$
	namespace string
}
`

var structTemplateNonNamespaced = `
// $.type|allLowercasePlural$ implements $.type|public$Interface
type $.type|allLowercasePlural$ struct {
	client $.pb$.$.ProtoType$ServiceClient
	$.Converter|raw$
	logger $.logr|raw$
}
`

var constructorTemplateNamespaced = `
// new$.type|publicPlural$ returns a $.type|allLowercasePlural$
func new$.type|publicPlural$(c *$.GroupGoName$$.Version$Client, namespace string) *$.type|allLowercasePlural$ {
	return &$.type|allLowercasePlural${
		client:    $.NewServiceClient$(c.ClientConn()),
		Converter: &$.Converter|raw$Impl{},
		logger:    c.logger.WithName("$.type|allLowercasePlural$"),
		namespace: namespace,
	}
}
`

var constructorTemplateNonNamespaced = `
// new$.type|publicPlural$ returns a $.type|allLowercasePlural$
func new$.type|publicPlural$(c *$.GroupGoName$$.Version$Client) *$.type|allLowercasePlural$ {
	return &$.type|allLowercasePlural${
		client: $.NewServiceClient$(c.ClientConn()),
		Converter: &$.Converter|raw$Impl{},
		logger: c.logger.WithName("$.type|allLowercasePlural$"),
	}
}
`

var structHelpers = `
func (c *$.type|allLowercasePlural$) getNamespace() *string {
	ns := ""
	if c == nil {
		return &ns 
	}
	return $if .namespaced$c.namespace$else$&ns$end$
}

func (c *$.type|allLowercasePlural$) fromProto(p *$.pb$.$.ProtoType$) *$.type|raw$ {
	if p == nil {
		return nil
	}
	obj := c.FromProto(p)
	obj.Default()
	return obj
}

func (c *$.type|allLowercasePlural$) fromProtoList(p *$.pb$.$.ProtoType$List) *$.type|raw$List {
	if p == nil {
		return nil
	}
	obj := &$.type|raw$List{
		ListMeta: c.FromProtoListMeta(p.ListMeta),
	}
	for _, item := range p.Items {
		obj.Items = append(obj.Items, *c.fromProto(item))
	}
	return obj
}
`

var listTemplate = `
func (c *$.type|allLowercasePlural$) List(ctx $.context|raw$, opts $.ListOptions|raw$) (*$.type|raw$List, error) {
	if opts.FieldSelector != "" {
		return nil, $.NewBadRequest|raw$("selectors are not supported for this resource")
	}
	if opts.Limit > 0 || opts.Continue != "" {
        return nil, $.NewBadRequest|raw$("pagination (limit/continue) is not supported for this resource")
    }

	resp, err := c.client.List$.ProtoType$s(ctx, &$.pb$.List$.ProtoType$sRequest{
		Namespace: c.getNamespace(),
		Opts:      c.ToProtoListOptions(&opts),
	})
	if err != nil {
		return nil, $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", "")
	}

	list := c.FromProtoList(resp.Get$.ProtoType$List())
	c.logger.V(5).Info("Listed $.type|public$s",
		"namespace", c.getNamespace(),
		"count", len(list.Items),
		"resource-version", list.GetResourceVersion(),
	)

	return list, nil
}
`

var getTemplate = `
func (c *$.type|allLowercasePlural$) Get(ctx $.context|raw$, name string, opts $.GetOptions|raw$) (*$.type|raw$, error) {
	resp, err := c.client.Get$.ProtoType$(ctx, &$.pb$.Get$.ProtoType$Request{
		Name:      name,
		Namespace: c.getNamespace(),
		Opts:      c.ToProtoGetOptions(&opts),
	})
	if err != nil {
		return nil, $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", name)
	}

	obj := c.fromProto(resp.Get$.ProtoType$())
	c.logger.V(6).Info("Fetched $.type|public$",
		"name", name,
		"namespace", c.getNamespace(),
		"uid", obj.GetUID(),
		"resource-version", obj.GetResourceVersion(),
	)

	return obj, nil
}
`

var deleteTemplate = `
func (c *$.type|allLowercasePlural$) Delete(ctx $.context|raw$, name string, opts $.DeleteOptions|raw$) error {
	if opts.GracePeriodSeconds != nil {
		return $.NewBadRequest|raw$("gracePeriodSeconds is not supported for this resource")
	}
	if opts.OrphanDependents != nil {
		return $.NewBadRequest|raw$("orphanDependents is not supported for this resource")
	}
	if opts.PropagationPolicy != nil {
		return $.NewBadRequest|raw$("propagationPolicy is not supported for this resource")
	}
	if len(opts.DryRun) != 0 {
		return $.NewBadRequest|raw$("dryRun is not supported for this resource")
	}

	_, err := c.client.Delete$.ProtoType$(ctx, &$.pb$.Delete$.ProtoType$Request{
		Name:      name,
		Namespace: c.getNamespace(),
		Opts:      c.ToProtoDeleteOptions(&opts), 
	})
	if err != nil {
		return $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", name)
	}

	c.logger.V(2).Info("Deleted $.type|public$",
		"name", name,
		"namespace", c.getNamespace(),
	)

	return nil
}
`

var createTemplate = `
func (c *$.type|allLowercasePlural$) Create(ctx $.context|raw$, $.type|allLowercase$ *$.type|raw$, opts $.CreateOptions|raw$) (*$.type|raw$, error) {
	if len(opts.DryRun) != 0 {
		return nil, $.NewBadRequest|raw$("dryRun is not supported for this resource")
	}

	if opts.FieldManager != "" {
		c.logger.V(6).Info("Ignoring unsupported create option", "option", "fieldManager")
	}
	if opts.FieldValidation != "" {
		c.logger.V(6).Info("Ignoring unsupported create option", "option", "fieldValidation")
	}

	resp, err := c.client.Create$.ProtoType$(ctx, &$.pb$.Create$.ProtoType$Request{
		$.ProtoType$: c.ToProto($.type|allLowercase$),	
		Opts:         &$.pb$.CreateOptions{},
	})
	if err != nil {
		return nil, $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", $.type|allLowercase$.GetName())
	}

	obj := c.fromProto(resp)
	c.logger.V(2).Info("Created $.type|public$",
		"name", obj.GetName(),
		"namespace", c.getNamespace(),
		"uid", obj.GetUID(),
		"resource-version", obj.GetResourceVersion(),
	)

	return obj, nil
}
`

var updateTemplate = `
func (c *$.type|allLowercasePlural$) Update(ctx $.context|raw$, $.type|allLowercase$ *$.type|raw$, opts $.UpdateOptions|raw$) (*$.type|raw$, error) {
	if len(opts.DryRun) != 0 {
		return nil, $.NewBadRequest|raw$("dryRun is not supported for this resource")
	}

	if opts.FieldManager != "" {
		c.logger.V(6).Info("Ignoring unsupported update option", "option", "fieldManager")
	}
	if opts.FieldValidation != "" {
		c.logger.V(6).Info("Ignoring unsupported update option", "option", "fieldValidation")
	}

	resp, err := c.client.Update$.ProtoType$(ctx, &$.pb$.Update$.ProtoType$Request{
		$.ProtoType$: c.ToProto($.type|allLowercase$),	
		Opts:         c.ToProtoUpdateOptions(&opts),
	})
	if err != nil {
		return nil, $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", $.type|allLowercase$.GetName())
	}

	obj := c.fromProto(resp)
	c.logger.V(2).Info("Updated $.type|public$",
		"name", obj.GetName(),
		"namespace", c.getNamespace(),
		"uid", obj.GetUID(),
		"resource-version", obj.GetResourceVersion(),
	)

	return obj, nil
}
`

var updateStatusTemplate = `
func (c *$.type|allLowercasePlural$) UpdateStatus(ctx $.context|raw$, $.type|allLowercase$ *$.type|raw$, opts $.UpdateOptions|raw$) (*$.type|raw$, error) {
	if len(opts.DryRun) != 0 {
		return nil, $.NewBadRequest|raw$("dryRun is not supported for this resource")
	}

	if opts.FieldManager != "" {
		c.logger.V(6).Info("Ignoring unsupported update option", "option", "fieldManager")
	}
	if opts.FieldValidation != "" {
		c.logger.V(6).Info("Ignoring unsupported update option", "option", "fieldValidation")
	}

	resp, err := c.client.Update$.ProtoType$Status(ctx, &$.pb$.Update$.ProtoType$StatusRequest{
		$.ProtoType$: c.ToProto($.type|allLowercase$),
		Opts:         c.ToProtoUpdateOptions(&opts),
	})
	if err != nil {
		return nil, $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", $.type|allLowercase$.GetName())
	}

	obj := c.fromProto(resp)
	c.logger.V(2).Info("Updated $.type|public$ status",
		"name", obj.GetName(),
		"namespace", c.getNamespace(),
		"uid", obj.GetUID(),
		"resource-version", obj.GetResourceVersion(),
	)

	return obj, nil
}
`

var watchTemplate = `
func (c *$.type|allLowercasePlural$) Watch(ctx $.context|raw$, opts $.ListOptions|raw$) ($.watchInterface|raw$, error) {
	if opts.FieldSelector != "" {
        return nil, $.NewBadRequest|raw$("selectors are not supported for this resource")
    }
	if opts.Limit > 0 || opts.Continue != "" {
        return nil, $.NewBadRequest|raw$("pagination (limit/continue) is not supported for this resource")
    }

	if opts.SendInitialEvents != nil && *opts.SendInitialEvents == true {
		if opts.ResourceVersionMatch != v1.ResourceVersionMatchNotOlderThan {
			return nil, apierrors.NewBadRequest("resourceVersionMatch must be NotOlderThan when sendInitialEvents is true")
		}
	}

	c.logger.V(4).Info("Opening watch stream",
		"watch", opts.Watch,
		"allowWatchBookmarks", opts.AllowWatchBookmarks,
		"resource", "$.type|allLowercasePlural$",
		"namespace", c.getNamespace(),
		"resource-version", opts.ResourceVersion,
		"timeout-seconds", opts.TimeoutSeconds,
		"sendInitialEvents", func(b *bool) any {
			if b == nil { return "nil" }
			return *b
		}(opts.SendInitialEvents),
	)

	ctx, cancel := context.WithCancel(ctx)
	stream, err := c.client.Watch$.ProtoType$s(ctx, &$.pb$.Watch$.ProtoType$sRequest{
		Namespace: c.getNamespace(),
		Opts:      c.ToProtoListOptions(&opts),
	})
	if err != nil {
		cancel()
		return nil, $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", "")
	}

	return $.nvgrpc|raw$(&$.type|allLowercasePlural$StreamAdapter{stream: stream, Converter: c.Converter}, cancel, c.logger), nil
}
`

var watchAdapterTemplate = `
// $.type|allLowercasePlural$StreamAdapter wraps the $.type|public$ gRPC stream to provide events.
type $.type|allLowercasePlural$StreamAdapter struct {
	stream $.pb$.$.ProtoType$Service_Watch$.ProtoType$sClient
	$.Converter|raw$
}

func (a *$.type|allLowercasePlural$StreamAdapter) Next() (string, $.runtime|raw$, error) {
	resp, err := a.stream.Recv()
	if err != nil {
		return string(watch.Error), nil, err
	}

	obj := a.FromProto(resp.GetObject())

	return resp.GetType(), obj, nil
}

func (a *$.type|allLowercasePlural$StreamAdapter) Close() error {
	return a.stream.CloseSend()
}
`

var patchTemplate = `
func (c *$.type|allLowercasePlural$) Patch(ctx $.context|raw$, name string, pt $.PatchType|raw$, data []byte, opts $.PatchOptions|raw$, subresources ...string) (*$.type|raw$, error) {
	if len(opts.DryRun) != 0 {
		return nil, $.NewBadRequest|raw$("dryRun is not supported for this resource")
	}

	if opts.Force != nil {
		c.logger.V(6).Info("Ignoring unsupported patch option", "option", "force")
	}
	if opts.FieldManager != "" {
		c.logger.V(6).Info("Ignoring unsupported patch option", "option", "fieldManager")
	}
	if opts.FieldValidation != "" {
		c.logger.V(6).Info("Ignoring unsupported patch option", "option", "fieldValidation")
	}

	resp, err := c.client.Patch$.ProtoType$(ctx, &$.pb$.Patch$.ProtoType$Request{
		Name: name,
		Namespace: c.getNamespace(),
		PatchType: string(pt),
		Data: data,
		Opts:         c.ToProtoPatchOptions(&opts),
		Subresources: subresources,
	})
	if err != nil {
		return nil, $.NewAPIError|raw$(err, "$.type|allLowercasePlural$", name)
	}

	obj := c.fromProto(resp)
	c.logger.V(2).Info("Patched $.type|public$",
		"name", name,
		"namespace", c.getNamespace(),
		"uid", obj.GetUID(),
		"resource-version", obj.GetResourceVersion(),
	)

	return obj, nil
}
`
