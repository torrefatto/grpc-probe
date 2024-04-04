#!/usr/bin/zsh

APP=${APP:-grpc-probe}
REGIONS=(fra par sin tyo sfo was)
if ! [ -z ${KOYEB_CONFIG} ]; then
    EXTRA_FLAGS=("-c" "${KOYEB_CONFIG}")
fi

for i in {1..${#REGIONS}}; do
    SVC=${REGIONS[$i]}
    OTHERS=()
    for OTHER in ${REGIONS[@]}; do
        if [ "${OTHER}" != "${SVC}" ]; then
            OTHERS+=("${OTHER}.${APP}.koyeb")
        fi
    done
    PEERS="${(j|;|)OTHERS}"

    echo "[creating] SVC -> ${SVC} --- PEERS -> ${PEERS}"

    koyeb ${EXTRA_FLAGS} services create \
        --app ${APP} \
        --git github.com/torrefatto/grpc-probe \
        --git-builder docker \
        --env "NAME=${SVC}.${APP}.koyeb" \
        --env "PEERS='${PEERS}'" \
        --port 12345 \
        --port 9090:http \
        --routes /${SVC}:9090 \
        --region "${SVC}" \
        --type web \
        "${SVC}"
done
