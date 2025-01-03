package gw_date

import (
	"strconv"
	"time"
)

type EraJapanese struct {
	Name  string
	Start time.Time
}

var (
	jstZone = time.FixedZone("JST", 9*60*60)
	eraList = []EraJapanese{
		{Name: "令和", Start: time.Date(2019, time.May, 1, 0, 0, 0, 0, jstZone)},
		{Name: "平成", Start: time.Date(1989, time.January, 8, 0, 0, 0, 0, jstZone)},
		{Name: "昭和", Start: time.Date(1926, time.December, 25, 0, 0, 0, 0, jstZone)},
		{Name: "大正", Start: time.Date(1912, time.July, 30, 0, 0, 0, 0, jstZone)},
		{Name: "明治", Start: time.Date(1868, time.October, 23, 0, 0, 0, 0, jstZone)},
	}
)

func FormatJapaneseEraYear(targetTime *time.Time) (string, int, error) {
	for _, era := range eraList {
		if era.Start.Before(*targetTime) {
			//初年度は１を返す
			return era.Name, targetTime.Year() - era.Start.Year() + 1, nil
		}
	}
	return "", 0, gw_errors.New("明治より前の元号には対応していません")
}

// 例：令和1年12月3日
func FormatJapaneseEraYYYYMD(targetTime *time.Time) (string, error) {
	for _, era := range eraList {
		if era.Start.Before(*targetTime) {
			//初年度は１を返す
			return era.Name + strconv.Itoa(targetTime.Year()-era.Start.Year()+1) + "年" + targetTime.Format("1月2日"), nil
		}
	}
	return "", gw_errors.New("明治より前の元号には対応していません")
}

func GetLastDayOfMonth(targetTime time.Time, location *time.Location) time.Time {
	firstOfMonth := time.Date(targetTime.Year(), targetTime.Month(), 1, 0, 0, 0, 0, location)
	firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)
	lastOfMonth := firstOfNextMonth.Add(-time.Duration(1) * time.Second)
	return lastOfMonth
}
func GetFirstDayOfMonth(targetTime time.Time, location *time.Location) time.Time {
	firstOfMonth := time.Date(targetTime.Year(), targetTime.Month(), 1, 0, 0, 0, 0, location)
	return firstOfMonth
}
func GetLastDayOfNextMonth(targetTime time.Time, location *time.Location) time.Time {
	firstOfMonth := time.Date(targetTime.Year(), targetTime.Month(), 1, 0, 0, 0, 0, location)
	firstOf2MonthAfter := firstOfMonth.AddDate(0, 2, 0)
	lastOfNextMonth := firstOf2MonthAfter.Add(-time.Duration(1) * time.Second)
	return lastOfNextMonth
}
func GetFirstDayOfNextMonth(targetTime time.Time, location *time.Location) time.Time {
	firstOfMonth := time.Date(targetTime.Year(), targetTime.Month(), 1, 0, 0, 0, 0, location)
	firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)
	return firstOfNextMonth
}

// 当月末を取得
func TruncDateAndGetLastSecondOfMonth(targetTime time.Time) (time.Time, time.Time) {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	truncatedDate := targetTime.In(jst)
	truncatedDate = truncatedDate.Truncate(time.Hour).Add(-time.Duration(truncatedDate.Hour()) * time.Hour)
	//今月末を取得
	toDate := GetLastDayOfMonth(truncatedDate, jst)
	return truncatedDate, toDate
}

// 月末の更新日（月末日 AM 9:00）を考慮して、最終日当日(AM0:00 以降)の場合は、翌月末を更新日に設定する
// return truncateDate, toDate, trialEndDate
func TruncDateAndGetTrialEndAndToDate(targetTime time.Time) (time.Time, time.Time, *time.Time) {
	truncatedDate, toDate := TruncDateAndGetLastSecondOfMonth(targetTime)
	if truncatedDate.Format("20060102") == toDate.Format("20060102") {
		//toDateを翌月末まで伸ばし、trial endはその月の月初とする
		jst, _ := time.LoadLocation("Asia/Tokyo")
		trialEnd := GetFirstDayOfNextMonth(toDate, jst)
		return truncatedDate,
			GetLastDayOfNextMonth(toDate, jst),
			&trialEnd
	} else {
		return truncatedDate, toDate, nil
	}
}

// X日後の日本時間ラスト秒を取得
func GetLastSecondOfDay(targetTime *time.Time, addDays int) *time.Time {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	truncatedDate := targetTime.In(jst)
	truncatedDate = truncatedDate.Truncate(time.Hour).Add(-time.Duration(truncatedDate.Hour()) * time.Hour)
	//今月末を取得
	truncatedDate = truncatedDate.AddDate(0, 0, addDays+1)
	retDate := truncatedDate.Add(-time.Duration(1) * time.Second)
	return &retDate
}

func MaxTime(a *time.Time, b *time.Time) *time.Time {
	if a == nil {
		return b
	} else if b == nil {
		return a
	}
	if a.After(*b) {
		return a
	} else {
		return b
	}
}
func MinTime(a *time.Time, b *time.Time) *time.Time {
	if a == nil {
		return b
	} else if b == nil {
		return a
	}
	if a.Before(*b) {
		return a
	} else {
		return b
	}
}

func GetYoubi(t time.Time) string {
	wdays := [...]string{"日", "月", "火", "水", "木", "金", "土"}
	return wdays[t.Weekday()]
}
