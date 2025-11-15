package models

import "encoding/json"

// SimilarityGroup 相似度组
type SimilarityGroup struct {
	// 组标识
	GroupID       string `json:"group_id"`       // 组唯一ID
	RepresentFile string `json:"represent_file"` // 代表文件URL

	// 成员信息
	Members     []SimilarityMember `json:"members"`      // 组成员
	MemberCount int                `json:"member_count"` // 成员数量

	// 相似度信息
	AvgSimilarity float64 `json:"avg_similarity"` // 平均相似度
	MinSimilarity float64 `json:"min_similarity"` // 最小相似度
	MaxSimilarity float64 `json:"max_similarity"` // 最大相似度

	// 去重建议
	DuplicateFiles []string `json:"duplicate_files"` // 建议删除的重复文件
	TotalSavedSize int64    `json:"total_saved_size"` // 去重后节省空间(字节)
}

// SimilarityMember 相似度组成员
type SimilarityMember struct {
	FileURL    string  `json:"file_url"`    // 文件URL
	FilePath   string  `json:"file_path"`   // 本地路径
	FileSize   int64   `json:"file_size"`   // 文件大小
	Similarity float64 `json:"similarity"`  // 与代表文件的相似度
}

// SimilarityMatrix 相似度矩阵
type SimilarityMatrix struct {
	Files  []string    `json:"files"`  // 文件URL列表
	Matrix [][]float64 `json:"matrix"` // NxN相似度矩阵
}

// ToJSON 序列化为JSON
func (s *SimilarityGroup) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
