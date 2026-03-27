rsync -avzhp laptop1:/home/myindo/workspace/myindo/hyperledger-demo/organizations/peerOrganizations/org2.example.com/ /home/valos/workspace/myindo/hlf-demo/organizations/peerOrganizations/org2.example.com/
sudo chown -R valos:valos /home/valos/workspace/myindo/hlf-demo/organizations/peerOrganizations/org2.example.com/



# after running 004-generate-configtx in central pc
rsync -avzhp /home/valos/workspace/myindo/hlf-demo/config/channel-artifacts/ laptop1:/home/myindo/workspace/myindo/hyperledger-demo/config/channel-artifacts/


rsync -avzhp /home/valos/workspace/myindo/hlf-demo/organizations/peerOrganizations/org1.example.com/ laptop1:/home/myindo/workspace/myindo/hyperledger-demo/organizations/peerOrganizations/org1.example.com/
