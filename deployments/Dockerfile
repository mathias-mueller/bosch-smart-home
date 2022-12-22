FROM python:3.9

RUN python3 -m pip install --user ansible

COPY ansible-hosts /etc/ansible/hosts
WORKDIR /build
COPY . /build

ENTRYPOINT ["tail", "-f", "/dev/null"]