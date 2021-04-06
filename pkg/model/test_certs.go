package model

var ValidPublicKey string = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDCnGXRhf571U4APnZmae/3O0Zxtis/d32sGRbwwZe/Zkon3/qCV3PS5zh2Uw9Cgq6q15zq6P4vNCMBgWEiu7nh/EUTn9bFyEnnhVZIEQ53qaNuaV8laNhYuV/qgypnLlhdm3dREKFWNm3aqkkVFoZjGv9lIjIMR4ioOW6lA9MsOYzNnQ5U5B66fVMNvMY/DQppCzlkffYkZa6ioa7meZXp03Mg5ywnoGlapzP0TxvmoN+yN+/POMZOVHKqbmiZZDPMMZnhOtFgreTMQG/9qDZytJke7o2ny840lXEUqdCQdWaXC0hlsTgQuK8tNiad02ryaDfLDaE3TVuQOSjIO3GbTftiPOSzSLzdWhrPkGLwGINGbT//No+vv3ezaqS7251ZLhIUzhvP+dQ9HEbQpFuYRJSByVzbvo9yqKTpoeK1OAlbFLDtVXEwpQKVOS4DUXa4vKD3VJbwJzA5dmNEehK2G+9nAvxm5LijSHwsmb3hTjJaIpoEh3WGD4wZBVl8pbE= test@waarp"
var InvalidPublicKey string = "public key"
var ValidPrivateKey string = `-----BEGIN OPENSSH PRIVATE KEY-----
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
-----END OPENSSH PRIVATE KEY-----`
var InvalidPrivateKey []byte = []byte("private key")
