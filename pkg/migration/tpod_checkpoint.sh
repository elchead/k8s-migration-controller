export PODNAME=${WORKER}
export JOBSIZE=S #`echo ${PODNAME} | cut -d- -f3`
NODE=`kubectl get pod  -o=jsonpath='{.spec.nodeName}' -n ${NS} ${WORKER}`
if [ "${NODE}" = "zone2" ]
then
NODE=zone3
elif [ "${NODE}" = "zone3" ]
then
NODE=zone2
fi
echo ${NODE}
export NODE=${NODE}
envsubst < pod_checkpoint.yaml | kubectl apply -f -
