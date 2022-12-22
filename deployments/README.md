# Deployment using Ansible

## 0. Install ansible

https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html#pip-install

## 1. Add servers to deploy to your inventory

Add hosts to `inventory.yaml`

See: https://docs.ansible.com/ansible/latest/user_guide/intro_inventory.html

## 2. Check if servers are reachable

```shell
ansible -i inventory.yaml all --list-hosts

ansible -i inventory.yaml all --ask-pass -u <user> -m ping

```

## 3. Create configuration

In `resources/config.ini` create  a valid configuration.
See `config.ini.default` in the root directory for an example.

In `resources/message.txt` define the telegram message to be sent out.

## 3. Build & Deploy service

```shell
ansible-playbook -i inventory.yaml --ask-pass -u <user> main.yml
```