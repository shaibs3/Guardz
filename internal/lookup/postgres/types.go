package postgres

// GORM models for demonstration
// (You can move these to a shared db package if you wish)
type GormPath struct {
	ID   uint      `gorm:"primaryKey"`
	Path string    `gorm:"uniqueIndex"`
	URLs []GormURL `gorm:"foreignKey:PathID"`
}

func (GormPath) TableName() string {
	return "paths"
}

type GormURL struct {
	ID     uint `gorm:"primaryKey"`
	PathID uint
	URL    string
}

func (GormURL) TableName() string {
	return "urls"
}
