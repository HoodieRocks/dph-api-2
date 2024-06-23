package utils

import (
	"time"
)

type User struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	Bio      string    `json:"bio"`
	Badges   []string  `json:"badges"`
	Icon     *string   `json:"icon"`
	JoinDate time.Time `json:"join_date"`
	Password string    `json:"-"`
	Token    string    `json:"-"`
}

type Project struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	Author        string     `json:"author"`
	Description   string     `json:"description"`
	Body          string     `json:"body"`
	Creation      time.Time  `json:"creation"`
	Updated       time.Time  `json:"updated"`
	Status        string     `json:"status"`
	Downloads     int        `json:"downloads"`
	Category      []string   `json:"category"`
	Icon          *string    `json:"icon"`
	License       *string    `json:"license"`
	FeaturedUntil *time.Time `json:"featured_until"`
}

type Version struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Creation     time.Time `json:"creation"`
	Downloads    int       `json:"downloads"`
	DownloadLink string    `json:"download_link"`
	VersionCode  string    `json:"version_code"`
	Supports     []string  `json:"supports"`
	Project      string    `json:"project"`
	RpDownload   *string   `json:"rp_download,omitempty"`
}
