package utils

import (
	"compress/gzip"
	"encoding/gob"
	"log"
	"os"
	"regexp"
	"time"
	"tx-tracker/pkg/models"
)

func ConvertTimestamp(unixTime int) string {

	timeStamp := time.Unix(int64(unixTime), 0)
	return timeStamp.String()
}

func Load[T comparable](filename string, toLoad *Set[T]) error {

	fi, err := os.Open(filename)
	if err != nil {
		noFile := regexp.MustCompile("(no such file or directory)")
		notFound := noFile.Find([]byte(err.Error()))
		if notFound == nil {
			log.Println(err)
			return err
		} else {
			fi, err = os.Create(filename)
			if err != nil {
				log.Println(err)
				return err
			}
			fi.Close()
			return nil
		}
	}
	defer fi.Close()
	fileInfo, err := fi.Stat()
	if fileInfo.Size() == 0 {
		return nil
	}
	if err != nil {
		log.Fatal(err)
	}
	fz, err := gzip.NewReader(fi)
	if err != nil {
		return err
	}
	defer fz.Close()

	decoder := gob.NewDecoder(fz)
	rawDic := make(map[T]string, 0)
	err = decoder.Decode(&rawDic)
	if err != nil {
		return err
	}
	for key, value := range rawDic {
		toLoad.Add(key, value)
	}

	return nil
}

func Save[T comparable](filename string, toSave *Set[T]) error {
	fi, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fi.Close()

	fz := gzip.NewWriter(fi)
	defer fz.Close()

	encoder := gob.NewEncoder(fz)
	err = encoder.Encode(&toSave.dic)
	if err != nil {
		return err
	}

	return nil
}

func RemoveOldItems(toCheck *Set[models.WatchTx], unixTimeNow int64) {
	for _, key := range toCheck.Keys() {
		twoWeeks := time.Unix(key.TimeRequested, 0).UTC().AddDate(0, 0, 14).Unix()
		if twoWeeks < unixTimeNow {
			toCheck.Remove(key)
		}
	}
}
