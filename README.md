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
- include an additional container to the redis master pod which run an applicaion to constantly back-up Redis DB file 
- write Go application to frequently send the copy of the Redis DB file to Google (object) [Storage](https://cloud.google.com/storage/)
- create a new Docker image to run backup application.

## How to use

Large part of effort to reproduce this project would be spend to prepare the environment. I will just refer to the external sources which has a detailed explanation of the preparation steps. Follow these steps in order.

### Get initial Guestbook runnning

Follow steps in the original [Guestbook](https://cloud.google.com/container-engine/docs/guestbook) and make sure everything works as it should. For your reference, I included several files (see [for-reference](./for-refernce) folder) from the Guestbook in this repository. Basically, the only file missing is redis-master-pod.json since we need to run redis-master with replication controller (not a single pod).

## Under the hood


## Feedback

