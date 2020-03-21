import subprocess
import json

completedProc = subprocess.run(["aws", "ec2", "describe-addresses"], capture_output=True)
associations = json.loads(completedProc.stdout)
addresses = associations['Addresses']

def is_minecraft_addr(addr):
    if addr['Tags'] is None:
        return False
    tags = addr['Tags']
    tags = [tag for tag in tags if tag['Key'] == 'purpose']
    if len(tags) != 1:
        return False

    return tags[0]['Value'] == "minecraft"

mc_addr = [addr for addr in addresses if is_minecraft_addr(addr)][0]
print(json.dumps(mc_addr))
