package repository

import "gorm.io/gorm"

// TxManager 为 service 层提供事务边界：情绪写入与回访联动必须在同一事务内完成。
type TxManager struct{ db *gorm.DB }

func NewTxManager(db *gorm.DB) *TxManager { return &TxManager{db: db} }

func (m *TxManager) Run(fn func(tx *gorm.DB) error) error {
	return m.db.Transaction(fn)
}
