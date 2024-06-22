package ruleengine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Rule represents a single rule.
type Rule struct {
	Field    string      `json:"field"`
	Type     string      `json:"type"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// RulesGroup represents a group of rules.
type RulesGroup struct {
	Condition string        `json:"combinator"`
	Rules     []interface{} `json:"rules"`
}

// EvaluateRule evaluates a single rule against the provided data.
func EvaluateRule(data map[string]interface{}, rule Rule) bool {	

	fieldValue, ok := data[rule.Field]
	fmt.Println("I am here in Evaluvate Rule" , fieldValue ,ok,  rule.Field)

	if !ok {
		return false
	}

	switch rule.Type {
	case "string":

		return evaluateStringRule(fieldValue.(string), rule ,  data)
	case "integer":
		return evaluateIntegerRule(fieldValue.(float64), rule ) // JSON numbers are decoded as float64
	case "date":
		return evaluateDateRule(fieldValue.(string), rule) // Assume date is a string
	}

	return false
}

func evaluateStringRule(fieldValue string, rule Rule, data map[string]interface{}) bool {
	value := rule.Value.(string)
	switch rule.Operator {
	case "=":
		return fieldValue == value
	case "!":
		return fieldValue != value
	case "contains":
		
		return strings.Contains(fieldValue, value)
	case "not_contains":
		return !strings.Contains(fieldValue, value)
	case "beginsWith":
		return strings.HasPrefix(fieldValue, value)
	case "endsWith":
		return strings.HasSuffix(fieldValue, value)
	case "doesNotContain":
		return !strings.Contains(fieldValue, value)
	case "doesNotBeginWith":
		return !strings.HasPrefix(fieldValue, value)
	case "doesNotEndWith":
		return !strings.HasSuffix(fieldValue, value)
	}
	
	return false
}

func evaluateIntegerRule(fieldValue float64, rule Rule) bool {
	value, err := strconv.Atoi(rule.Value.(string))
	if err != nil {
		return false
	}
	fieldValueInt := int(fieldValue)
	switch rule.Operator {
	case "equal":
		return fieldValueInt == value
	case "not_equal":
		return fieldValueInt != value
	case "less":
		return fieldValueInt < value
	case "greater":
		return fieldValueInt > value

	}
	return false
}

func evaluateDateRule(fieldValue string, rule Rule) bool {
	value, err := time.Parse("2006-01-02 15:04:05", rule.Value.(string))
	if err != nil {
		return false
	}
	fieldDate, err := time.Parse("2006-01-02 15:04:05", fieldValue)
	if err != nil {
		return false
	}
	switch rule.Operator {
	case "equal":
		return fieldDate.Equal(value)
	case "not_equal":
		return !fieldDate.Equal(value)
	case "less":
		return fieldDate.Before(value)
	case "greater":
		return fieldDate.After(value)
	}
	return false
}

// EvaluateRulesGroup evaluates a group of rules against the provided data.
func EvaluateRulesGroup(data map[string]interface{}, group RulesGroup) bool {

	result := group.Condition == "and"
	for _, ruleInterface := range group.Rules {
		switch rule := ruleInterface.(type) {
		case map[string]interface{}:
			var r Rule
			ruleBytes, _ := json.Marshal(rule)
			json.Unmarshal(ruleBytes, &r)
			if group.Condition == "and" {
				result = result && EvaluateRule(data, r)
			} else if group.Condition == "or" {
				result = result || EvaluateRule(data, r)
			}
		case RulesGroup:
			if group.Condition == "and" {
				result = result && EvaluateRulesGroup(data, rule)
			} else if group.Condition == "or" {
				result = result || EvaluateRulesGroup(data, rule)
			}
		}
	}
	return result
}
