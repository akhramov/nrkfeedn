package main

//go:generate oapi-codegen --package=playback -generate=client,types --exclude-schemas="#/components/schemas/AvailabilityVm" -o ./playback/playback.gen.go https://psapi.nrk.no/documentation/openapi/playback/openapi.json
//go:generate oapi-codegen --package=psapi -generate=client,types -o ./psapi/psapi.gen.go --import-mapping=../playback/openapi.json:github.com/akhramov/fattigmanns_nrk_radio/playback https://psapi.nrk.no/documentation/openapi/programsider-radio/openapi.yml

import (
	"net/http"

	"github.com/akhramov/fattigmanns_nrk_radio/service"

	"github.com/labstack/echo/v4"
)

func main() {
	s, err := service.New()
	if err != nil {
		panic(err)
	}

	e := echo.New()

	e.GET("/:id", func(c echo.Context) error {
		res, err := s.GetFeed(c.Param("id"))
		if err != nil {
			c.XML(http.StatusBadRequest, err)
		}

		return c.Blob(http.StatusOK, "application/rss+xml", []byte(res))
	})

	e.POST("/:id", func(c echo.Context) error {
		err := s.CreatePodcast(c.Param("id"))
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		return c.String(http.StatusOK, "ok")
	})

	e.Logger.Fatal(e.Start(":8084"))
}
