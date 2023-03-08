// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package htmlutil

import (
	"golang.org/x/net/html"
)

func VisitHTML(node *html.Node, before, after func(*html.Node) error) error {
	if before != nil {
		if err := before(node); err != nil {
			return err
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := VisitHTML(child, before, after); err != nil {
			return err
		}
	}
	if after != nil {
		if err := after(node); err != nil {
			return err
		}
	}
	return nil
}

func GetAttr(node *html.Node, namespace, name string) (val string, ok bool) {
	if node == nil {
		return "", false
	}
	for _, attr := range node.Attr {
		if attr.Namespace == namespace && attr.Key == name {
			return attr.Val, true
		}
	}
	return "", false
}
