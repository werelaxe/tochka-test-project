package main

import (
	"errors"
	"regexp"
)

type CompiledRule struct {
	TitlePattern       regexp.Regexp
	ItemPattern        regexp.Regexp
	LinkPattern        regexp.Regexp
	DescriptionPattern regexp.Regexp
}

func CompileRule(rule *Rule) (*CompiledRule, error) {
	compiledItemPattern, err := regexp.Compile(rule.ItemPattern)
	if err != nil {
		return nil, errors.New("compilation rule error: " + err.Error())
	}
	compiledTitlePattern, err := regexp.Compile(rule.TitlePattern)
	if err != nil {
		return nil, errors.New("compilation rule error: " + err.Error())
	}
	compiledLinkPattern, err := regexp.Compile(rule.LinkPattern)
	if err != nil {
		return nil, errors.New("compilation rule error: " + err.Error())
	}
	compiledDescriptionPattern, err := regexp.Compile(rule.DescriptionPattern)
	if err != nil {
		return nil, errors.New("compilation rule error: " + err.Error())
	}
	result := CompiledRule{
		TitlePattern:       *compiledTitlePattern,
		DescriptionPattern: *compiledDescriptionPattern,
		ItemPattern:        *compiledItemPattern,
		LinkPattern:        *compiledLinkPattern,
	}
	return &result, nil
}
