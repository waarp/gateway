Configuration SFTP
##################

L'objet JSON de configuration du protocole SFTP est identique pour les serveurs
et les partenaires. Sa structure est la suivante :

* **address** (*string*) - L'adresse du serveur SFTP (local ou distant).
* **port** (*number*) - Le port utilisé par le serveur SFTP.
* **keyExchanges** (*array* of *string*) - Liste des algorithmes d'échange de
  clé utilisées par le serveur. Voir ci-dessous pour les valeurs autorisées.
  Par défaut, tous les algorithmes sont autorisés.
* **ciphers** (*array* of *string*) - Liste des algorithmes de cryptage
  symétrique des données. Voir ci-dessous pour les valeurs autorisées.
  Par défaut, tous les algorithmes sont autorisés.
* **macs** (*array* of *string*) - Liste des algorithmes d'authentification de
  message (MAC). Voir ci-dessous pour les valeurs autorisées.
  Par défaut, tous les algorithmes sont autorisés.


**Algorithmes d'échange de clé**

- ``diffie-hellman-group1-sha1``
- ``diffie-hellman-group14-sha1``
- ``ecdh-sha2-nistp256``
- ``ecdh-sha2-nistp384``
- ``ecdh-sha2-nistp521``
- ``curve25519-sha256@libssh.org``

**Algorithmes de cryptage symétrique**

- ``aes128-gcm@openssh.com``
- ``aes128-ctr``
- ``aes192-ctr``
- ``aes256-ctr``
- ``chacha20-poly1305@openssh.com``

**Algorithmes d'authentification de message**

- ``hmac-sha2-256-etm@openssh.com``
- ``hmac-sha2-256``
- ``hmac-sha1``
- ``hmac-sha1-96``