package services

import (
	"net"

	"github.com/oschwald/maxminddb-golang"
	"github.com/sirupsen/logrus"
	"github.com/thiagozs/geolocation-go/models"
)

type MaxMindDB struct {
	db  *maxminddb.Reader
	log *logrus.Entry
}

func NewMindMax(log *logrus.Entry) (*MaxMindDB, error) {
	dbLocation := "db/GeoLite2-City.mmdb"
	log.Printf("Opening database: %s\n", dbLocation)

	db, err := maxminddb.Open(dbLocation)
	if err != nil {
		return &MaxMindDB{}, err
	}
	return &MaxMindDB{db, log}, nil
}

func (m *MaxMindDB) Close() error {
	return m.db.Close()
}

func (m *MaxMindDB) Lookup(ip net.IP) (models.Record, error) {
	var record models.Record

	err := m.db.Lookup(ip, &record)
	if err != nil {
		return record, err
	}

	record.IP = ip.String()
	return record, nil
}
