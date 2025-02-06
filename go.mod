module github.com/generalworksinc/goutil

go 1.22

require (
	github.com/dsnet/compress v0.0.1
	github.com/getsentry/sentry-go v0.31.1
	github.com/google/uuid v1.6.0
	github.com/morikuni/failure/v2 v2.0.0-20240419002657-2551069d1c86
	github.com/oklog/ulid/v2 v2.0.2
	github.com/yeka/zip v0.0.0-20180914125537-d046722c6feb
	golang.org/x/crypto v0.31.0
	golang.org/x/text v0.21.0
)

require (
	github.com/ktnyt/go-moji v1.0.0 // indirect
	github.com/kurehajime/cjk2num v0.0.0-20210929142953-005d508333d0 // indirect
	golang.org/x/sys v0.28.0 // indirect
)

replace github.com/generalworksinc/goutil => ./
