// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package yamlmeta

import (
	"fmt"
)

type NodeType interface {
	Name() string
}

type Schema interface {
	AssignType(node Node)
	//AllowedTypes() []*NodeType
}

type AnySchema struct {
}

var _ Schema = &AnySchema{}

type DocumentSchema struct {
	Name    string
	Source  *Document
	Allowed *DocumentType
}

var _ Schema = &DocumentSchema{}

type MapSchema struct {
	Allowed *MapType
}

var _ Schema = &MapSchema{}

type MapItemSchema struct {
	Allowed *MapItemType
}

var _ Schema = &MapItemSchema{}

type ValueSchema struct {
	Allowed ValueType
}

func (v ValueSchema) AssignType(node Node) {
	panic("implement me")
}

var _ Schema = &ValueSchema{}

type TypeCheck struct {
	Violations []string
}

func (tc *TypeCheck) HasViolations() bool {
	return len(tc.Violations) > 0
}

type DocumentType struct {
	Source *Document
	ValueSchema Schema // MapSchema, ArraySchema, ... ScalarSchema
}

type ValueType struct {
	DefaultValue interface{}
}

type MapType struct {
	Items []*MapItemType
}

type MapItemType struct {
	Key         interface{}
	ValueSchema Schema
}

type ArrayType struct {
	Array
}

func NewDocumentSchema(doc *Document) (*DocumentSchema, error) {
	schemaDoc := &DocumentSchema{Allowed: &DocumentType{&Document{Value: nil}}}

	switch typedContent := doc.Value.(type) {
	case *Map:
		mapSchema, err := NewMapSchema(typedContent)
		if err != nil {
			return nil, err
		}
		schemaDoc.Allowed.Schema = mapSchema
	case *Array:
		return nil, NewArraySchema()
	}

	return schemaDoc, nil
}

func NewMapSchema(m *Map) (*MapSchema, error) {
	mapType := &MapType{}

	for _, mapItem := range m.Items {
		newMapItemSchema, err := NewMapItemType(mapItem)
		if err != nil {
			return nil, err
		}
		mapType.Items = append(mapType.Items, newMapItemSchema)
	}
	return &MapSchema{Allowed: mapType}, nil
}

func NewMapItemType(schemaItem *MapItem) (*MapItemType, error) {
	switch typedContent := schemaItem.Value.(type) {
	case *Map:
		newMapSchema, err := NewMapSchema(typedContent)
		if err != nil {
			return nil, err
		}
		return &MapItemType{Key: schemaItem.Key, Schema: newMapSchema}, nil
	case string:
		return &MapItemType{Key: schemaItem.Key, Schema: ValueSchema{Allowed: ValueType{DefaultValue: schemaItem.Value}}}, nil
	case int:
		return &MapItemType{Key: schemaItem.Key, Schema: ValueSchema{Allowed: ValueType{DefaultValue: schemaItem.Value}}}, nil
	case *Array:
		return nil, NewArraySchema()
	}
	return nil, fmt.Errorf("Map Item type did not match any know types")
}

func NewArraySchema() error {
	return fmt.Errorf("Arrays are currently not supported in schema")
}

func (mt *MapType) AllowsKey(key interface{}) bool {
	for _, item := range mt.Items {
		if item.Key == key {
			return true
		}
	}
	return false
}

func (mt MapType) CheckAllows(item *MapItem) TypeCheck {
	typeCheck := TypeCheck{}

	if !mt.AllowsKey(item.Key) {
		typeCheck.Violations = append(typeCheck.Violations, fmt.Sprintf("Map item '%s' at %s is not defined in schema", item.Key, item.Position.AsCompactString()))
	}
	return typeCheck
}

func (d *Document) Check() TypeCheck {
	var typeCheck TypeCheck

	switch typedContents := d.Value.(type) {
	case Node:
		typeCheck = typedContents.Check()
	}

	return typeCheck
}

func (m *Map) Check() TypeCheck {
	typeCheck := TypeCheck{}

	for _, item := range m.Items {
		check := m.Type.CheckAllows(item)
		if check.HasViolations() {
			typeCheck.Violations = append(typeCheck.Violations, check.Violations...)
			continue
		}
		check = item.Check()
		if check.HasViolations() {
			typeCheck.Violations = append(typeCheck.Violations, check.Violations...)
		}
	}
	return typeCheck
}

func (mapItem *MapItem) Check() TypeCheck {
	typeCheck := TypeCheck{}
	violationErrorMessage := "Map item '%s' at %s was type %T when %T was expected."

	switch typedValue := mapItem.Value.(type) {
	case *Map:
		check := typedValue.Check()
		typeCheck.Violations = append(typeCheck.Violations, check.Violations...)
	default:
		if ok := mapItem.Type.(typedValue); !ok {
		//if _, ok := mapItem.Type.(int); !ok {
			violation := fmt.Sprintf(violationErrorMessage, mapItem.Key, mapItem.Position.AsCompactString(), typedValue, mapItem.Schema.AllowedTypes())
			typeCheck.Violations = append(typeCheck.Violations, violation)
		}
		return typeCheck
	}

	return typeCheck
}

func (d *DocumentSet) Check() TypeCheck { return TypeCheck{} }
func (d *Array) Check() TypeCheck       { return TypeCheck{} }
func (d *ArrayItem) Check() TypeCheck   { return TypeCheck{} }

func (as AnySchema) AssignType(_ Node) {}

func (s DocumentSchema) AssignType(node Node) {
	document, ok := node.(*Document)
	if !ok {
		panic("document schema assigntype called with non-document type")
	}
	document.Type = s.Allowed

	for _, val := range node.GetValues() {
		switch docVal := val.(type) {
		case Node:
			s.Allowed.ValueSchema.AssignType(docVal)
		default:
			//if !s.Allowed.ValueSchema.IsAssignable(val) {}
			return
		}
	}
}

func (m MapSchema) AssignType(node Node) {
	mapNode, ok := node.(*Map)
	if !ok {
		panic("map schema assigntype called with non-map type")
	}
	mapNode.Type = m.Allowed

	for _, mapItem := range mapNode.Items {
		for _, itemType := range mapNode.Type.Items {
			if mapItem.Key == itemType.Key {
				// Found the matching schema section
				itemType.ValueSchema.AssignType(mapItem)
				break
			}
		}

	}
}

func (m MapItemSchema) AssignType(node Node) {
	mapItemNode, ok := node.(*MapItem)
	if !ok {
		panic("mapItem schema assigntype called with non-mapItem type")
	}

	mapItemNode.Type = m.Allowed

	switch mapItemVal := mapItemNode.Value.(type) {
	case Node:
		mapItemNode.Type.ValueSchema.AssignType(mapItemVal)
	default:
		return
	}
}
