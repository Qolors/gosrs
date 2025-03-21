package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/qolors/gosrs/internal/db"
	"github.com/qolors/gosrs/internal/osrsclient"
)

var (
	dbi *db.DBInstance
)

func InitDBService(ctx context.Context) error {
	if dbi == nil {
		var err error
		dbi, err = db.InitializeConnection(ctx)
		return err
	}

	return nil
}

func PullAll() (osrsclient.PullAllHistory, error) {
	query := `
		SELECT polled_at, skills, activities
		FROM public.user_stats
		ORDER BY polled_at DESC LIMIT 100
	`

	rows, err := dbi.Conn.Query(context.Background(), query)
	if err != nil {
		return osrsclient.PullAllHistory{}, err
	}
	defer rows.Close()

	var history osrsclient.PullAllHistory

	for rows.Next() {
		var (
			polledAt   time.Time
			skillsJSON []byte
			actJSON    []byte
		)

		if err := rows.Scan(&polledAt, &skillsJSON, &actJSON); err != nil {
			return osrsclient.PullAllHistory{}, err
		}

		var skills []osrsclient.Skill
		if err := json.Unmarshal(skillsJSON, &skills); err != nil {
			return osrsclient.PullAllHistory{}, fmt.Errorf("skills unmarshal error: %w", err)
		}

		var activities []osrsclient.Activity
		if err := json.Unmarshal(actJSON, &activities); err != nil {
			return osrsclient.PullAllHistory{}, fmt.Errorf("activities unmarshal error: %w", err)
		}

		item := osrsclient.PullAllItem{
			TimeStamp:  polledAt,
			Skills:     skills,
			Acitivites: activities,
		}

		history.Items = append(history.Items, item)
	}

	if err = rows.Err(); err != nil {
		return osrsclient.PullAllHistory{}, err
	}

	return history, nil
}

func InsertPolling(skillsJSON []byte, activitiesJSON []byte) error {

	query := "INSERT INTO user_stats (polled_at, skills, activities) VALUES ($1, $2::jsonb, $3::jsonb);"
	_, err := dbi.Conn.Exec(context.Background(), query, time.Now().UTC(), skillsJSON, activitiesJSON)

	return err
}

func CloseConnection() {
	if dbi != nil {
		dbi.Conn.Close()
	}
}
