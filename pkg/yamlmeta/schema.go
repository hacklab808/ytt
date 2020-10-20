// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package yamlmeta

import (
	"fmt"
)

type Schema interface {
	AssignType(typeable Typeable)
}

var _ Schema = &AnySchema{}
var _ Schema = &DocumentSchema{}

type AnySchema struct {
}

type DocumentSchema struct {
	Name    string
	Source  *Document
	Allowed *DocumentType
}

type ScalarType struct {
	Name string
}

func NewDocumentSchema(doc *Document) (*DocumentSchema, error) {
	docType := &DocumentType{Source: doc}

	switch typedDocumentValue := doc.Value.(type) {
	case *Map:
		valueType, _ := NewMapType(typedDocumentValue)

		docType.ValueType = valueType
	}
	return &DocumentSchema{
		Name:    "dataValues",
		Source:  doc,
		Allowed: docType,
	}, nil
}

func NewMapType(m *Map) (*MapType, error) {
	mapType := &MapType{}

	for _, mapItem := range m.Items {
		mapItemType, err := NewMapItemType(mapItem)
		if err != nil {
			return nil, err
		}
		mapType.Items = append(mapType.Items, mapItemType)
	}
	return mapType, nil
}

func NewMapItemType(item *MapItem) (*MapItemType, error) {
	switch typedContent := item.Value.(type) {
	case *Map:
		mapType, err := NewMapType(typedContent)
		if err != nil {
			return nil, err
		}
		return &MapItemType{Key: item.Key, ValueType: mapType}, nil
	case string:
		return &MapItemType{Key: item.Key, ValueType: ScalarType{Name: "string"}}, nil
	case int:
		return &MapItemType{Key: item.Key, ValueType: ScalarType{Name: "int"}}, nil
	case *Array:
		return nil, NewArraySchema()
	}
	return nil, fmt.Errorf("Map Item type did not match any know types")
}

func NewArraySchema() error {
	return fmt.Errorf("Arrays are currently not supported in schema")
}

func (as *AnySchema) AssignType(_ Node) {}

func (s *DocumentSchema) AssignType(doc *Document) TypeCheck {
	return s.Allowed.ValueType.AssignTypeTo(doc)
}