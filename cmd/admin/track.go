package admin

import (
	"encoding/json"
	"fmt"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/admind"
	"github.com/freetocompute/kebe/pkg/admind/requests"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"io/ioutil"
	"strconv"
	"time"
)

var snapName string
var trackName string

func init() {
	track.AddCommand(addTrack)
	addTrack.Flags().StringVarP(&snapName, "snap-name", "s", "", "The name of the snap to add a track to")
	addTrack.Flags().StringVarP(&trackName, "track-name", "t", "", "The name of the track to add")
	_ = addTrack.MarkFlagRequired("snap-name")
	_ = addTrack.MarkFlagRequired("track-name")
}

var track = &cobra.Command{
	Use:   "track",
	Short: "track",
}

var addTrack = &cobra.Command{
	Use:   "add",
	Short: "add",
	Run: func(cmd *cobra.Command, args []string) {
		var loginInfo admind.LoginInfo
		bytes, _ := ioutil.ReadFile(LoginConfigFilename)
		err := json.Unmarshal(bytes, &loginInfo)
		if err != nil {
			panic(err)
		}

		// we've expired and we need to refresh, so for now
		// force a login
		if time.Now().After(loginInfo.Token.Expiry) {
			fmt.Printf("Token has expired, refreshing.\n")
			loginInfoPtr, err := refreshToken(loginInfo.Token.RefreshToken)
			if err != nil {
				panic(err)
			}

			if loginInfoPtr != nil {
				loginInfo = *loginInfoPtr
			}
		}

		addTrackReq := requests.AddTrack{
			SnapName:  snapName,
			TrackName: trackName,
		}

		// Create a Resty Client
		client := resty.New()

		admindURL := config.MustGetString(configkey.AdminDURL)

		trackURL := admindURL + "/v1/admin/track"

		bytes, _ = json.Marshal(&addTrackReq)
		resp, err := client.R().
			SetBody(bytes).
			SetHeader("Authorization", loginInfo.Token.AccessToken).
			Post(trackURL)

		if err != nil {
			panic(err)
		}

		if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
			if resp.Error() != nil {
				panic(resp.Error())
			}
			panic("there was a problem: " + strconv.Itoa(resp.StatusCode()))
		}
	},
}
