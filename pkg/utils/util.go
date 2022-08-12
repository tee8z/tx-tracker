package utils

import "time"

func ConvertTimestamp(unixTime int) string {

	timeStamp := time.Unix(int64(unixTime), 0)
	return timeStamp.String()
}
