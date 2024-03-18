/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"errors"
	"net/http"
	"regexp"
	"encoding/json"
	"io"
	"log"
	"bytes"
	"strconv"
	"os"


	"github.com/spf13/cobra"
	"github.com/manifoldco/promptui"
	"github.com/joho/godotenv"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an album to rating database",
	Long: `
Use this command to add an album to the rating database.
It will first prompt you to enter a the uri of a spotify album
	- Then checks if it is actually an album URI
	- Then strips the URI to an ID
	- Then it will display the album properties and allow you to confirm
Then it will ask you for a rating
	- Then checks if it is an integer 0-10
Then it will push to mongo database.
`,
	Run: func(cmd *cobra.Command, args []string) {
		addAlbum()
	},
}

func addAlbum() {
	id := getAlbumURI()// get the album id

	album, err := getAlbumData(id)// get the album data
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}

	rating, err := getRating()// get the rating
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}

	errr := pushToMongo(album, rating, id)

	if errr != nil {
		fmt.Printf("Error! %v\n", errr)
	} else {
		fmt.Printf("%v added!\n", album.AlbumName)
	}

	
	// push to mongo
}
//spotify:album:0HmKhR7Umt3ACs52ZLnKyK

func pushToMongo(albumP *Album, rating int, id string) error {
	err := godotenv.Load()
	if err != nil {
	  log.Fatal("Error loading .env file")
	}
	MONGO := os.Getenv("MONGO")
	album := *albumP
	combinedData := map[string]interface{}{
		"albumName":  album.AlbumName,
		"artistName": album.ArtistName,
		"spotifyURI": id,
		"img":        album.Image,
		"link":       album.Link,
		"rating":     rating,
	}
	
	data, err := json.Marshal(combinedData)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	//fmt.Println("JSON Data:", string(data))

	resp, err := http.Post(MONGO, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//fmt.Printf("code: %v\n", resp.StatusCode)

	return nil
}

func getAlbumURI() string {
	pattern := `^spotify:album:[A-Za-z0-9]{22}$` //uri regex pattern

	re := regexp.MustCompile(pattern) // create the regex

	validate := func(input string) error { // validating function to validate the input with the regex
		if re.MatchString(input) == false {
			return errors.New("Invalid uri")
		}	
		return nil
	}

	prompt := promptui.Prompt{ // get the spotify uri with the regex validating
		Label:    "Enter spotify URI",
		Validate: validate,
	}

	result, err := prompt.Run() // actually run the prompt

	if err != nil { // if there was an error, print the error
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	id := result[14:] // since we only reach here if the pattern matches (i think), now we splice only the id
	return id
}
//`https://personal-site-production-2d9f.up.railway.app/api/album/
func getAlbumData(id string) (*Album, error) {
	albumToAdd := getAlbumAPICall(id) //get the album from api call
	ap := &albumToAdd
	fmt.Printf("\n%v by %v\n", albumToAdd.AlbumName, albumToAdd.ArtistName)
	confirm := promptui.Prompt{ // confirmation function
		Label: "Is this the album you would like to rate?",
		IsConfirm: true,
	}

	confirmation, err := confirm.Run()

	if err != nil { // if there is an error
		fmt.Printf("Prompt failed %v\n", err)
		return &Album{}, err
	}
	if confirmation == "y"{ // if we are confirmed
		fmt.Printf("\n Okay time to rate")
		return ap, nil
	}
	return &Album{}, err
}

type AccessTokenResponse struct {
	Token string `json:"access_token"`
}

func getAccessToken(ch chan<- string){
	err := godotenv.Load()
	if err != nil {
	  log.Fatal("Error loading .env file")
	}
	ACCESS := os.Getenv("ACCESS")
	resp, err := http.Post(ACCESS, "application/json", nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var responseObj AccessTokenResponse
	json.Unmarshal(body, &responseObj)

	accessToken := responseObj.Token

	ch<- accessToken
}

type Album struct {
	AlbumName string `json:"albumName"`
	ArtistName string `json:"artistName"`
	Image string `json:"img"`
	Link string `json:"link"`
}

type SpotifyAlbumRequest struct {
	ID string `json:"spotifyURI"`
	Token string `json:"access_token"`
}

func getAlbumAPICall(id string) Album {
	err := godotenv.Load()
	if err != nil {
	  log.Fatal("Error loading .env file")
	}
	ALBUM := os.Getenv("ALBUM")
	ch := make(chan string)
	go getAccessToken(ch)
	accessToken := <- ch

	// now we get the album, so first we should make an album struct
	request := SpotifyAlbumRequest{ID: id, Token: accessToken}
	data, err := json.Marshal(request)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := http.Post(ALBUM, "application/json", 
		bytes.NewBuffer(data))
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var responseObj Album
	json.Unmarshal(body, &responseObj)

	//albName := responseObj.AlbumName

	//fmt.Printf("\nAlbum Name: %v\nArtist Name: %v\nImage link: %v\nURL: %v\n", responseObj.AlbumName, responseObj.ArtistName, responseObj.Image, responseObj.Link)

	return responseObj
}

func getRating() (int, error) {
	validate := func(input string) error { 
		rating, err := strconv.Atoi(input)

		if err != nil {
			return errors.New("Not an integer!")
		}

		if rating <= 0 || rating > 10 {
			return errors.New("Not an integer 1-10")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Enter a rating 1-10",
		Validate: validate,
	}

	rating, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
	}

	ratingInt, nonErr := strconv.Atoi(rating)

	if nonErr != nil {
		fmt.Printf("%v\n", nonErr)
	}

	return ratingInt, err
}



func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
