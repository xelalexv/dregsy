FROM scratch
ADD dregsy /
CMD ["/dregsy", "-config=config.yaml"]
