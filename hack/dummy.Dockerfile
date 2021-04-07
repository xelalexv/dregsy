FROM busybox

RUN dd if=/dev/urandom of=/random.dummy bs=4M count=1
