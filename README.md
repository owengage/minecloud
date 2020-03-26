# Minecraft on AWS EC2

Code for running Minecraft servers on EC2. Currently has lots of hard coded
peices that makes it difficult for anyone but myself to use.

# Log

## 2020-03-26

I need to make the set-up do a lot more. It needs to trigger a download of the
world, as well as all the other pieces that are involved such as the server 
properties.

## 2020-03-25

Sorted out some of the IAM stuff for AWS. I can now make an EC2 instance that
has access to ECR and S3.

Having trouble remotely setting up the instance however. I can SSH via Go but

1. I don't know the public key ahead of time.
2. I need to install docker which then requires me to log out and SSH back in
   again.
3. The AWS CLI on the default instance seems to be old, and logging in to the
   container registry is different to on my own box's CLI. Wants me to use 
   `aws ecr get-login` rather than `aws ecr get-login-password`. The former 
   seems deprecated.

I also need to rethink how the server wrapper works. I don't really want to
tar up all of the server properties and ops and things. I'll probably want
easy access to those in future.

# TODOs

## All cloud

At the moment I store a few things on my own computer to make things easy. At
the moment I store the elastic IP details and the instance ID of the EC2
instance that is currently running. It would be better to use tags on the
appropriate resources instead. This means that I would be able to provision and
decommision servers with no dependency on the machine that they were run on.
The client should be as stateless as I can get.

Tags I'll likely want:

* minecraft-server: would be the string-name of the minecraft server. At the
  moment I've got this hard-coded as just 'cliff-side'. This name would tell me
  where to locate the s3 object and elastic IP.

I actually can't see what other ones I would want. If I tag the elastic IP and
instance with these names then that should be all the information I need.

## Codify template

The EC2 launch template is just stored on my AWS account. Being able to create
it from this project would be useful. Or maybe not use the templates at all if
it's going to be programatic anyway.

## EBS?

Is EBS a better way to store servers? Will it introduce more latency?

## Wrap server

It would be useful if I could programatically send commands to the minecraft
server. It looks like the STDIN for the server becomes the server console.
But I run the server as a daemon, so there is no STDIN. It might make sense to
wrap the server in another executable that is fit to be a daemon, that exposes
some way to communicate with the server. A Go application that accepts TCP
connections over localhost would suffice, or maybe UNIX sockets?

Update: Ended up doing this via TCP and wrapping the whole lot in a docker
image rather than making a daemon. I can then just use docker to daemonise it
on the EC2 instance. This means I can still access logs from it if I want to.
