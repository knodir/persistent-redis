Persistent Redis master for the Guestbook 
================

TL; DR -- Make Kubernetes Guestbook's Redis database persistent by using Google Storage.

## Introduction

Google Cloud Platform's [Guestbook](https://cloud.google.com/container-engine/docs/guestbook) application is a great place to start playing with [Kubernetes](https://github.com/googlecloudplatform/kubernetes). Once everything starts working (as it should be), one can easily scale up/down web servers and Redis workers to match increasing/decreasing client requests. Such scalability, availability, and automation is a product of Kubernetes' services, pods, replication controllers, and of course [Docker containers](https://github.com/docker/docker). 

## Problem

However, Guestbook tutorial does not consider availability of the Redis master pod, which runs container with the only copy of the Redis database. When that container (or redis master pod) goes down, it would be impossible to store any of guest entries. This problem can be easily solved by running redis master via replication controller, i.e., similar to redis workers, but with replica value equal to one.

Redis master replication controller solves availability problem, but not the data loss. Because pods are stateless, everything on previously running container will be lost (including Redis DB file). Replication controller creates a fresh replica of the Redis master, which does not have any entries made to the original master. Ideally, we would like replication controller to create Redis master with all previous guest entries. This project shows one way of doing it, using Google Storage.

## Approach

There are at least three different solutions to this problem, which are listed below: 
- use Docker's data volume containers (a.k.a. data-only containers). Tom Offermann has a great [post](http://www.offermann.us/2013/12/tiny-docker-pieces-loosely-joined.html) how to do this (solves different problem)
- mount [Google persistent disks](https://cloud.google.com/compute/docs/disks) to container and store Redis DB file on that disk
- run another container to constantly save snapshot of the DB file in reliable storage.

Since I was planning to hack Kubernetes broader, I decided to go with the last option. Thus, I could play not only with Docker volumes, but also Dockerfiles, pods, write Go applicaiton to backup DB file, and dockerize that application. I ended up doing followings:
- create a replication controller for the redis master pod
- include an additional container to the redis master pod which runs applicaion to constantly back-up Redis DB file 
- write Go application to frequently send the copy of the Redis DB file to Google (object) [Storage](https://cloud.google.com/storage/)
- create a new Docker image to run backup application.

## How to use

Large part of effort to reproduce this project would be spend to prepare the environment. I will just refer to the external sources which has a detailed explanation of the preparation steps. Follow these steps in order.

### Get initial Guestbook running

Follow steps in the original [Guestbook](https://cloud.google.com/container-engine/docs/guestbook) and make sure everything works as it should. For your reference, I included several files (see [for-reference](./for-reference) folder) from the Guestbook in this repository. Basically, the only file missing is `redis-master-pod.json` since we need to run redis-master with replication controller (not a single pod).

### Run redis master with replication controller

Delete redis master pod from running Guestbook application via 

	gcloud preview container pods delete <master-pod-uuid> 

Run master controller via replication controller using

	gcloud preview container replicationcontrollers create \ 
	--config-file $CONFIG_DIR/redis-master-controller.json

intead of the following command in the original Guestbook

	gcloud preview container pods create \ 
	--config-file $CONFIG_DIR/redis-master-pod.json

When you do 

	gcloud preview container replicationcontrollers list 

you should see redis-master-controller running two images, knodir/redis and knodir/redis-backup with replicas equal to one. 

Similarly,

	gcloud preview container pods list 

should show two containers, with aforementioned images, running under the same pod.

Now to check if everything is working as expected, you can make some guest entries (on browser), delete a pod running redis master (or one of those containers by logging into host VM), and see replication controller create another pod with all previously entered guest posts. Congrats, if it did so! Otherwise, keep reading; I'll explain what is going on under the hood, which should be helpful for debugging. 


## Under the hood

As explained above, replication controller creates a redis master pod, which runs two containers, one to store Redis database (the same as original Guestbook) and the other to constantly backup Redis DB file to Google storage. You might have noticed I use knodir/redis image instead of `dockerfile/redis`. The only change I did to the original [dockerfile/redis](https://github.com/dockerfile/redis) image is to update Redis database configuration to take the snapshot of the DB file after each insert record. You can see that in line #22 of the modified [Dockerfile](https://github.com/knodir/persistent-redis/blob/master/redis/Dockerfile) in [redis](./redis) folder (compared to the original `dockerfile/redis` [Dockerfile](https://github.com/dockerfile/redis/blob/master/Dockerfile)).

	sed -i 's/save 900 1/save 3 1/' /etc/redis/redis.conf && \ 

This change lets us to see the result of our project right away, instead of waiting 15 minutes for Redis to write entries to DB file (default Redis behavior). Of course this is is too much overhead and never should be done in any useful deployment (except this educational one :). This project works fine with original `dockerfile/redis`, too.

`redis-master-controller.json` file defines shared folder `redis-bckp` which will be mounted in `/data` folder of the both containers. This is the same folder where Redis saved its snapshot file `dump.rdb` in the original `dockerfile/redis` image. I just made this folder [pod's volume](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/volumes.md), which will be accessible to all containers running in the same pod. The application running in another container, `knodir/redis-backup`, writes `dump.rdb` file to Google storage every 5 seconds (hardcoded, but possible to change by updating Docker image). Application also reads data from Google storage and replaces local `dump.rdb` at container boot time (only). This is the core idea of the Google storage based Redis data resilience approach.

The second container's image is `knodir/redis-backup` created using [google/golang](https://registry.hub.docker.com/u/google/golang/) image, which bundles container with the latest version of Golang. [Dockerfile](https://github.com/knodir/persistent-redis/blob/master/redis-backup/Dockerfile) in [redis-backup](https://github.com/knodir/persistent-redis/tree/master/redis-backup) folder shows how to make this image run our redis-backup application, `app.go`. Also, note that `app.go` uses my credentials to access Google storage services. That credentials are of my trial account, which will expire within a week (by Jan. 6, 2015). Although credentials are hardcoded, it is possible to update them with yours by replacing `cache.json` file via sign-up to [Google Storage](https://cloud.google.com/storage/).


## Feedback

This is a tiny project, mostly to educate myself about Kubernetes, so I don't expect much feedback. But if you get stuck reproducing this, solved the same problem with different approach, or anything relevant, feel free to open Github issue, or email me.
