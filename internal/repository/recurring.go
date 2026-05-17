package repository

import "github.com/amiraminb/coinwarrior/internal/model"

type recurringDocument struct {
	SchemaVersion  int                   `json:"schema_version"`
	RecurringRules []model.RecurringRule `json:"recurring_rules"`
}

func (r *FileRepository) LoadRecurringRules() ([]model.RecurringRule, error) {
	path, err := r.DataFilePath(RecurringFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d recurringDocument) []model.RecurringRule {
		return d.RecurringRules
	}, nil)
}

func (r *FileRepository) SaveRecurringRules(rules []model.RecurringRule) error {
	path, err := r.DataFilePath(RecurringFileName)
	if err != nil {
		return err
	}
	if rules == nil {
		rules = []model.RecurringRule{}
	}
	return saveDocument(path, recurringDocument{SchemaVersion: 1, RecurringRules: rules})
}
