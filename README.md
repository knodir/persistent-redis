Persistent Redis for the Guestbook 
================

TL; DR -- Make Kubernetes Guestbook's Redis database persistent by using Google Storage.

## Introduction

Google Cloud Platform's [Guestbook](https://cloud.google.com/container-engine/docs/guestbook) application is a great place to start playing with [Kubernetes](https://github.com/googlecloudplatform/kubernetes). Once everything starts working, one can easily scale up/down web servers and Redis workers to match increasing/decreasing client requests. Such scalability, availability, and automation is a product of Kubernetes' building blocks, such as services, pods, replication controllers, and of course [Docker containers](https://github.com/docker/docker). 

## Problem

However, Guestbook tutorial does not consider availability of the Redis master pod, which runs a container with the only copy of the Redis database. When this container (or redis master pod) goes down, it will be impossible to store guest entries made thereafter. This problem can be easily solved by running redis master via replication controller, similar to redis workers, but with replica value equal to one.

Running Redis master with replication controller solves availability problem, but does not prevent data loss. Because pods are stateless by construction, all data local to previous container will be lost (including Redis DB file). Replication controller creates a fresh replica of the Redis master pod, which even though provides the same functionality, does not have DB file of the failed container. Ideally, we would like the replication controller to create Redis master with all previous guest entries. This project shows one way of achieving this, using Google Storage.

## Approach

There are at least three different solutions to this problem as listed below: 
- use Docker's data volume containers (a.k.a. data-only containers). Tom Offermann has a great [post](http://www.offermann.us/2013/12/tiny-docker-pieces-loosely-joined.html) about this (solves different problem)
- mount [Google persistent disk](https://cloud.google.com/compute/docs/disks) to container and use it store Redis DB file
- run another container to constantly save snapshot of the DB file in reliable storage.

Since I was planning to have broader Kubernetes experience, I decided to go with the last option. Thus, I could hack not only Docker volumes, but also Dockerfiles, pods, write Go applicaiton to backup DB file, and dockerize that application. I ended up doing followings:
- create a replication controller for the redis master pod
- put an additional container to the redis master pod, which runs a code to constantly back-up Redis DB file 
- write Go application to send the copy of the Redis DB file to [Google Storage](https://cloud.google.com/storage/)
- create a new Docker image which runs backup application.

## How to use

Large part of time to reproduce this project will be spend to prepare the environment. I will just refer to the external sources, which have a detailed explanation of the preparation steps. These steps should be run in a given order.

### Get initial Guestbook up and running

Follow instructions in the original [Guestbook](https://cloud.google.com/container-engine/docs/guestbook) and make sure everything works as it should. For your reference, I included several files (see [for-reference](./for-reference) folder) from the original Guestbook. Basically, the only file missing is `redis-master-pod.json` since we need to run redis master with the replication controller (not as a standalone pod).

### Run redis master with replication controller

Once the original Guestbook is run without any problem, delete redis master pod from running Guestbook application via 

	gcloud preview container pods delete <master-pod-uuid> 

Create Redis master replication controller with

	gcloud preview container replicationcontrollers create \ 
	--config-file $CONFIG_DIR/redis-master-controller.json

intead of the following pod creation command in the original Guestbook

	gcloud preview container pods create \ 
	--config-file $CONFIG_DIR/redis-master-pod.json

When you do 

	gcloud preview container replicationcontrollers list 

you should see `redis-master-controller` running two images, `knodir/redis` and `knodir/redis-backup` with `Replica(s)` equal to one. Similarly,

	gcloud preview container pods list 

should show two containers, with aforementioned images, running under the same pod.

Now, to check if everything is working as expected, you can make some guest entries (on browser), delete a pod running redis master (or one of those containers by manually logging into the host VM), and see replication controller create another Redis master pod, with all previously entered guest posts. 

Congrats, if it did so! Otherwise, keep reading; I'll explain what is going on under the hood, which should be helpful for debugging. 


## Under the hood

As explained above, replication controller creates a redis master pod, which runs two containers, one to store Redis database (the same as original Guestbook), and the other to constantly backup Redis DB file to Google storage. You might have noticed `redis-master-controller.json` uses [knodir/redis](https://registry.hub.docker.com/u/knodir/redis/) image instead of [dockerfile/redis](https://github.com/dockerfile/redis). `knodir/redis` image is based on `dockerfile/redis` with the only one change shown line #22 of modified [Dockerfile](https://github.com/knodir/persistent-redis/blob/master/redis/Dockerfile) in [redis](./redis) folder (compared to the original `dockerfile/redis` [Dockerfile](https://github.com/dockerfile/redis/blob/master/Dockerfile)). 

	sed -i 's/save 900 1/save 3 1/' /etc/redis/redis.conf && \ 

This update changes Redis database configuration file to make the snapshot of the DB file after each DB record. Such change is rather for our testing convinience, it lets us to see the result DB file backup logic much faster, instead of Redis's default behavior to snapshot DB each 15 minutes. Of course this creates too much overhead, and never should be usef in any useful deployment (except this educational one :). This project works fine with original `dockerfile/redis`, too.

We use Kubernetes's [pod volume](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/volumes.md) feature to share data between two container. `redis-master-controller.json` file defines pod volume called `redis-bckp` which is mounted in `/data` folder of the both containers. This is the same folder where Redis saves its DB snapshot file `dump.rdb` in the original `dockerfile/redis` image. As pod volume is accessible to all containers running on the same pod, the application running in another container, `knodir/redis-backup`, writes `dump.rdb` file to Google storage every 5 seconds (hardcoded frequence, but possible to change by updating Docker image). This application also reads data from Google storage, and replaces local `dump.rdb` at container boot time (only). This is one of the core ideas of the Google storage based Redis data resilience approach.

The second container's image is [knodir/redis-backup](https://registry.hub.docker.com/u/knodir/redis-backup/) created using [google/golang](https://registry.hub.docker.com/u/google/golang/) image, which bundles container with the latest version of Golang. [Dockerfile](https://github.com/knodir/persistent-redis/blob/master/redis-backup/Dockerfile) in [redis-backup](https://github.com/knodir/persistent-redis/tree/master/redis-backup) folder shows how to make this image run our redis-backup application, `app.go`. Note that `app.go` uses my credentials to access Google storage services. These credentials are of my trial account, which will expire within a week (by Jan. 6, 2015). Although credentials are hardcoded, it is possible to substitute them with yours by replacing `cache.json` file via creating account in [Google Storage](https://cloud.google.com/storage/).


## Feedback

This is a tiny project, mostly to educate myself about Kubernetes, so I do not expect much feedback. But if you get stuck reproducing this, solve the same problem with different approach, or anything relevant, feel free to email me or open Github issue. 

Happy hacking!
