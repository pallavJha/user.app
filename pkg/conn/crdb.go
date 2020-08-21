package conn

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var Instance *sql.DB

func InitDBConnection() (*sql.DB, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=disable",
		viper.GetString("crdb.host"),
		viper.GetInt32("crdb.port"),
		viper.GetString("crdb.user"),
		viper.GetString("crdb.dbname"),
	)

	instance, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to create db connection")
	}

	instance.SetMaxOpenConns(viper.GetInt("crdb.max_open_conns"))
	instance.SetMaxIdleConns(viper.GetInt("crdb.max_idle_conns"))
	return instance, nil
}
