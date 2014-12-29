/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This code is adoptod from original Google Cloud Storage Go Sample Application, here
	https://github.com/GoogleCloudPlatform/storage-getting-started-go
*/

// Binary storage-sample creates a new bucket, performs all of its operations
// within that bucket, and then cleans up after itself if nothing fails along the way.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"io"
	"time"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/storage/v1"
)

const (
	// Change these variable to match your personal information.
	// bucketName   = "redis_store_temp"
	projectID    = "keen-mission-793"
	clientId     = "927316323145-h5fmb9ecnquckn17o18bta8llquu0gb8.apps.googleusercontent.com"
	clientSecret = "C0XLkKqczfjNgMXfjf3rYkGf"

	// For the basic sample, these variables need not be changed.
	scope      = storage.DevstorageFull_controlScope
	authURL    = "https://accounts.google.com/o/oauth2/auth"
	tokenURL   = "https://accounts.google.com/o/oauth2/token"
	entityName = "allUsers"
	redirectURL = "urn:ietf:wg:oauth:2.0:oob"
)

var (
	cacheFile = flag.String("cache", "cache.json", "Token cache file")
	code      = flag.String("code", "", "Authorization Code")

	bucketName   = "redis_master_bckp"
	fileName   = "test.txt" // The name of the local file to upload, e.g., /path/to/file.txt
	objectName = "master-data"    // This can be changed to any valid object name. e.g., master-data

	// For additional help with OAuth2 setup,
	// see http://goo.gl/cJ2OC and http://goo.gl/Y0os2

	// Set up a configuration boilerplate.
	config = &oauth.Config{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		Scope:        scope,
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		TokenCache:   oauth.CacheFile(*cacheFile),
		RedirectURL:  redirectURL,
	}
)

func main() {
	
	flag.Parse()

	// Set up a transport using the config
	transport := &oauth.Transport{
		Config:    config,
		Transport: http.DefaultTransport,
	}

	token, _ := config.TokenCache.Token()
	// token, err := config.TokenCache.Token()
	// if err != nil {
	// 	if *code == "" {
	// 		url := config.AuthCodeURL("")
	// 		fmt.Println("Visit URL to get a code then run again with -code=YOUR_CODE")
	// 		fmt.Println(url)
	// 		os.Exit(1)
	// 	}

	// 	// Exchange auth code for access token
	// 	token, err = transport.Exchange(*code)
	// 	if err != nil {
	// 		log.Fatal("[GS_BACKUP][FATAL]: Exchange: ", err)
	// 	}
	// 	log.Printf("[GS-BACKUP][INFO]: Token is cached in %v\n", config.TokenCache)
	// }

	transport.Token = token

	httpClient := transport.Client()
	service, _ := storage.New(httpClient)
	// service, err := storage.New(httpClient)
	is_acl_ok := false


	log.Printf("[GS-BACKUP]: read data from bucket: %s", bucketName)

	// Get ACL for an object.
	if res, err := service.ObjectAccessControls.Get(bucketName, objectName, entityName).Do(); err == nil {
		log.Printf("[GS-BACKUP][INFO]: Users in group %v can access %v/%v as %v.\n\n",
			res.Entity, bucketName, objectName, res.Role)
		is_acl_ok = true
	} else {
		// log.Fatalf("[GS-BACKUP]: Failed to get ACL for %s/%s: %v.\n", bucketName, objectName, err)
		log.Printf("[GS-BACKUP]: No bucket exist, yet. Proceed to write. %s/%s: %v.\n", bucketName, objectName, err)
	}

	if is_acl_ok == true {
		// Get an object from a bucket.
		if res, err := service.Objects.Get(bucketName, objectName).Do(); err == nil {
			log.Printf("[GS-BACKUP][INFO]: The media download link for %v/%v is %v.\n\n", bucketName, res.Name, res.MediaLink)
			response, e := http.Get(res.MediaLink)
			if (e != nil) {
				log.Fatalf("[GS-BACKUP][FATAL]: could not read data from: %s, error: %s", res.MediaLink, e)
			}
			defer response.Body.Close()

			// open file for writing
			file, err := os.Create("text.read")
			if (err != nil) {
				// fmt.Printf("ERROR: could not open file for write, error: %s", err)
				// return
				log.Fatalf("[GS-BACKUP][FATAL]: could not open file for write, error: %s", err)
			}

			// Use io.Copy to copy a file from URL to a locald disk
			_, err = io.Copy(file, response.Body)
			if (err != nil) {
				// fmt.Printf("ERROR: could not open file for write, error: %s", err)
				// return
				log.Fatalf("[GS-BACKUP][FATAL]: could not open file for write, error: %s", err)
			}
			file.Close()
			// fmt.Println("File successfully saved!")
			log.Printf("[GS-BACKUP][INFO]: File successfully saved!")

		} else {
			// fmt.Printf("Failed to get %s/%s: %s.\n", bucketName, objectName, err)
			log.Printf("[GS-BACKUP][INFO]: Failed to get %s/%s: %s.\n", bucketName, objectName, err)
		}
	}

	// If the bucket already exists and the user has access, warn the user, but don't try to create it.
	if _, err := service.Buckets.Get(bucketName).Do(); err == nil {
		// fmt.Printf("Bucket %s already exists - skipping buckets.insert call.\n", bucketName)
		log.Printf("[GS-BACKUP][INFO]: Bucket %s already exists - skipping buckets.insert call.\n", bucketName)
	} else {
		// Create a bucket.
		if res, err := service.Buckets.Insert(projectID, &storage.Bucket{Name: bucketName}).Do(); err == nil {
			// fmt.Printf("Created bucket %v at location %v\n\n", res.Name, res.SelfLink)
			log.Printf("[GS-BACKUP][INFO]: Created bucket %v at location %v\n\n", res.Name, res.SelfLink)
		} else {
			// fmt.Printf("Failed creating bucket %s: %v\n", bucketName, err)
			log.Printf("[GS-BACKUP][INFO]: Failed creating bucket %s: %v\n", bucketName, err)
		}
	}

	// Now keep backing up redis database file every 5 seconds
	for {
		// Insert an object into a bucket.
		object := &storage.Object{Name: objectName}
		file, err := os.Open(fileName)
		if err != nil {
			log.Fatalf("[GS-BACKUP][FATAL]: Error opening backup file %q: %v", fileName, err)
		}
		if res, err := service.Objects.Insert(bucketName, object).Media(file).Do(); err == nil {
			log.Printf("[GS-BACKUP][INFO]: Successfully backed up redis database as an object %v at location %v\n\n", res.Name, res.SelfLink)
		} else {
			// fmt.Printf("Objects.Insert failed: %v", err)
			log.Printf("[GS-BACKUP][FATAL]: Redis database file backup failed: %v", err)
		}
		file.Close()

		// Insert ACL for an object.
		// This illustrates the minimum requirements.
		objectAcl := &storage.ObjectAccessControl{
			Bucket: bucketName, Entity: entityName, Object: objectName, Role: "READER",
		}
		service.ObjectAccessControls.Insert(bucketName, objectName, objectAcl).Do();

		// if res, err := service.ObjectAccessControls.Insert(bucketName, objectName, objectAcl).Do(); err == nil {
		// 	fmt.Printf("Result of inserting ACL for %v/%v:\n%v\n\n", bucketName, objectName, res)
		// } else {
		// 	fmt.Printf("Failed to insert ACL for %s/%s: %v.\n", bucketName, objectName, err)
		// }

		time.Sleep(time.Second * 5)
	}
}
