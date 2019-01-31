#!/bin/sh

# Populate regisry host with disk images encapsulated inside container images. 
# Disk images are taken from /tmp/shared/images directory populated by cdi-func-test-registry-init
# Container images are built with buildah 

#images args
IMAGES_SRC=$1        #path to files to be encapsulated in docker image
IMAGES_CTR=$2        #path to directories with Dockerfile per file

#registry args
REGISTRY_HOST=$3     #host name of docker registry
REGISTRY_PORT=$4     #port of docker registry
REGISTRY_TLS="--tls-verify=false"

#health args
HEALTH_PATH=$5       #path f or health indicators - shared between popuplate and registry-host containers
HEALTH_PERIOD=$6
READYNESS_PATH=$7    #path f or readyness indicators - shared between popuplate and registry-host containers
READYNESS_PERIOD=$8

function  ready {
    echo "readiness"
    touch $1
}

function health {
    echo "health"
    touch $1
}

#Convert all images to docker build consumable format
DIR="-dir"
DOCKERFILE="Dockerfile"

function prepareImages {
   images_in=$1
   images_out=$2 
   
   rm -rf $images_out
   mkdir -p $images_out
   cp  $images_in* $images_out

   cd $images_out

   for FILENAME in $(ls); do
        mkdir -p $FILENAME$DIR
        cp  $FILENAME $FILENAME$DIR

        FILE=$FILENAME$DIR"/"$DOCKERFILE
        /bin/cat  >$FILE <<-EOF
                FROM scratch
                ADD / $FILENAME
EOF

        rm $FILENAME
  done
}

function error {
	if [ "$1" -ne "0" ]; then
           echo "Exiting on error"
	   exit -1
	fi
}

#Iterate over all images build them and push them into cdi registry
function pushImages {
   images=$1 
   registry_host=$2
   registry_port=$3
   registry_tls=$4
   registry=$registry_host":"$registry_port
   
   retval=$?
   shopt -s nullglob
   for IMAGEDIR in *$DIR; do
        cd $IMAGEDIR
        FILE=$(ls | grep -v $DOCKERFILE)
        declare -l FILE
        FILE=$FILE
        echo "building image "$FILE
        buildah bud -t $FILE":buildah" $images"/"$IMAGEDIR"/"; $retval
	error $retval	
        echo "pushing image "$FILE" to registry-service: "$resgistry
        buildah push $registry_tls  $FILE":buildah" "docker://"$registry"/"$FILE; $retVal
	error $retval
        cd ../
   done
}


#start health beat
health $HEALTH_PATH $HEALTH_PERIOD &

#prepare and poush images
prepareImages $IMAGES_SRC $IMAGES_CTR
pushImages  $IMAGES_CTR $REGISTRY_HOST $REGISTRY_PORT $REGISTRY_TLS

#mark container as ready
ready $READYNESS_PATH $READYNESS_PERIOD &

#sleep forever
trap : TERM INT
sleep infinity & wait


