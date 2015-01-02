Persistent Redis master for the Guestbook 
================

TL;DR -- Make Kubernetes Guestbook's Redis database persistent by using Google Storage.

## Introduction

Google Cloud Platform's [Guestbook](https://cloud.google.com/container-engine/docs/guestbook) application is a great place to start playing with [Kubernetes](https://github.com/googlecloudplatform/kubernetes). Once everything starts working (as it should be), one can easily scale up/down web servers and Redis workers to match increasing/decreasing client requests. Such scalability, availability, and automation is a product of Kubernetes' services, pods, replication controllers, and of course [Docker containers](https://github.com/docker/docker). 

## Problem

However, Guestbook tutorial does not consider availability of the Redis master pod, which runs container with the only copy of the Redis database. When that container (or redis master pod) goes down, it would be impossible to store any of guest entries. This problem can be easily solved by running redis master via replication controller, i.e., similar to redis workers, but with replica value equal to one.

Redis master replication controller solves availability problem, but not the data loss. Because pods are stateless, everything on previously running container will be lost (including Redis DB file). Replication controller creates a fresh replica of the Redis master, which does not have any entries made to the original master. Ideally, we would like replication controller to create Redis master with all previous guest entries. This project shows one way of doing it, using Google Storage.

## Approach

