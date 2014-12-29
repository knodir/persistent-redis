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
	"fmt"
	"log"
	"net/http"
	"os"
	"io"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/storage/v1"
)

const (
	// Change these variable to match your personal information.
	// bucketName   = "redis_store_temp"
	projectID    = "keen-mission-793"
	clientId     = "927316323145-h5fmb9ecnquckn17o18bta8llquu0gb8.apps.googleusercontent.com"
	clientSecret = "C0XLkKqczfjNgMXfjf3rYkGf"

	// fileName   = "/home/knodir/devel/k8s_workshop/google_storage/test.txt" // The name of the local file to upload.
	// objectName = "master-data"    // This can be changed to any valid object name.

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
	fileName   = "/home/knodir/devel/k8s_workshop/google_storage/test.txt" // The name of the local file to upload, e.g., /path/to/file.txt
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

func fatalf(service *storage.Service, errorMessage string, args ...interface{}) {
	restoreOriginalState(service)
	log.Fatalf("Dying with error:\n"+errorMessage, args...)
}

func restoreOriginalState(service *storage.Service) bool {
	succeeded := true

	// Delete an object from a bucket.
	if err := service.Objects.Delete(bucketName, objectName).Do(); err == nil {
		fmt.Printf("Successfully deleted %s/%s during cleanup.\n\n", bucketName, objectName)
	} else {
		// If the object exists but wasn't deleted, the bucket deletion will also fail.
		fmt.Printf("Could not delete object during cleanup: %v\n\n", err)
	}

	// Delete a bucket in the project
	if err := service.Buckets.Delete(bucketName).Do(); err == nil {
		fmt.Printf("Successfully deleted bucket %s during cleanup.\n\n", bucketName)
	} else {
		succeeded = false
		fmt.Printf("Could not delete bucket during cleanup: %v\n\n", err)
	}

	if !succeeded {
		fmt.Println("WARNING: Final cleanup attempt failed. Original state could not be restored.\n")
	}
	return succeeded
}

func main() {
	// flag.Parse()

	/* We expect to run this program as:
		go run gs-cmd.go -list -- to list existing Google storage buckets; 
		go run gs-cmd.go -read <bucket-name> -- to read data from <bucket-name>; 
		go run gs-cmd.go -write <bucket-name> -- to write data on bucket <bucket-name>; 
	*/
	listPtr := flag.Bool("list", false, "a bool")
	descPtr := flag.Bool("describe", false, "should be true to describe a bucket")
	readPtr := flag.Bool("read", false, "should be true to read bucket data")
	writePtr := flag.Bool("write", false, "should be true to write into bucket")

	// fmt.Println("listPtr:", *listPtr)
	// fmt.Println("descPtr:", *descPtr)
	// fmt.Println("read:", *readPtr)
	// fmt.Println("writePtr:", *writePtr)

	flag.Parse()

	// Set up a transport using the config
	transport := &oauth.Transport{
		Config:    config,
		Transport: http.DefaultTransport,
	}

	token, err := config.TokenCache.Token()
	if err != nil {
		if *code == "" {
			url := config.AuthCodeURL("")
			fmt.Println("Visit URL to get a code then run again with -code=YOUR_CODE")
			fmt.Println(url)
			os.Exit(1)
		}

		// Exchange auth code for access token
		token, err = transport.Exchange(*code)
		if err != nil {
			log.Fatal("Exchange: ", err)
		}
		fmt.Printf("Token is cached in %v\n", config.TokenCache)
	}
	transport.Token = token

	httpClient := transport.Client()
	service, err := storage.New(httpClient)

	switch {

	case *listPtr == true:
		fmt.Println("List all buckets")
		// List all buckets in a project.
		if res, err := service.Buckets.List(projectID).Do(); err == nil {
			fmt.Println("Buckets:")
			for _, item := range res.Items {
				fmt.Println(item.Id)
			}
			fmt.Println()
		} else {
			fatalf(service, "Buckets.List failed: %v", err)
		}

	case *descPtr == true:
		fmt.Println("list of objects in this bucket:", bucketName)

		// List all objects in a bucket
		if res, err := service.Objects.List(bucketName).Do(); err == nil {
			fmt.Printf("Objects in bucket %v:\n", bucketName)
			for _, object := range res.Items {
				fmt.Println(object.Name)
			}
			fmt.Println()
		} else {
			fmt.Println("No such bucket exist.")
			// fatalf(service, "Objects.List failed: %v", err)
		}

	case *readPtr == true:
		fmt.Println("read data from bucket:", bucketName)

		// Get ACL for an object.
		if res, err := service.ObjectAccessControls.Get(bucketName, objectName, entityName).Do(); err == nil {
			fmt.Printf("Users in group %v can access %v/%v as %v.\n\n",
				res.Entity, bucketName, objectName, res.Role)
		} else {
			// fatalf(service, "Failed to get ACL for %s/%s: %v.", bucketName, objectName, err)
			fmt.Printf("Failed to get ACL for %s/%s: %v.\n", bucketName, objectName, err)
		}

		// Get an object from a bucket.
		if res, err := service.Objects.Get(bucketName, objectName).Do(); err == nil {
			fmt.Printf("The media download link for %v/%v is %v.\n\n", bucketName, res.Name, res.MediaLink)
			response, e := http.Get(res.MediaLink)
			if (e != nil) {
				fmt.Printf("ERROR: could not read data from: %s, error: %s", res.MediaLink, e)
				return
			}
			defer response.Body.Close()

			// open file for writing
			file, err := os.Create("text.read")
			if (err != nil) {
				fmt.Printf("ERROR: could not open file for write, error: %s", err)
				return
			}

			// Use io.Copy to copy a file from URL to a locald disk
			_, err = io.Copy(file, response.Body)
			if (err != nil) {
				fmt.Printf("ERROR: could not open file for write, error: %s", err)
				return
			}
			file.Close()
			fmt.Println("File successfully saved!")

		} else {
			// fatalf(service, "Failed to get %s/%s: %s.", bucketName, objectName, err)
			fmt.Printf("Failed to get %s/%s: %s.\n", bucketName, objectName, err)
		}

	case *writePtr == true:
		fmt.Println("write to bucket:", bucketName)

		// If the bucket already exists and the user has access, warn the user, but don't try to create it.
		if _, err := service.Buckets.Get(bucketName).Do(); err == nil {
			fmt.Printf("Bucket %s already exists - skipping buckets.insert call.\n", bucketName)
		} else {
			// Create a bucket.
			if res, err := service.Buckets.Insert(projectID, &storage.Bucket{Name: bucketName}).Do(); err == nil {
				fmt.Printf("Created bucket %v at location %v\n\n", res.Name, res.SelfLink)
			} else {
				// fatalf(service, "Failed creating bucket %s: %v", bucketName, err)
				fmt.Printf("Failed creating bucket %s: %v\n", bucketName, err)
			}
		}

		// Insert an object into a bucket.
		object := &storage.Object{Name: objectName}
		file, err := os.Open(fileName)
		if err != nil {
			fatalf(service, "Error opening %q: %v", fileName, err)
		}
		if res, err := service.Objects.Insert(bucketName, object).Media(file).Do(); err == nil {
			fmt.Printf("Created object %v at location %v\n\n", res.Name, res.SelfLink)
		} else {
			fatalf(service, "Objects.Insert failed: %v", err)
		}

		// Insert ACL for an object.
		// This illustrates the minimum requirements.
		objectAcl := &storage.ObjectAccessControl{
			Bucket: bucketName, Entity: entityName, Object: objectName, Role: "READER",
		}
		if res, err := service.ObjectAccessControls.Insert(bucketName, objectName, objectAcl).Do(); err == nil {
			fmt.Printf("Result of inserting ACL for %v/%v:\n%v\n\n", bucketName, objectName, res)
		} else {
			// fatalf(service, "Failed to insert ACL for %s/%s: %v.", bucketName, objectName, err)
			fmt.Printf("Failed to insert ACL for %s/%s: %v.\n", bucketName, objectName, err)
		}

	default:
		fmt.Println("nothing: ...")	
	}

	// if !restoreOriginalState(service) {
	// 	os.Exit(1)
	// }

}
