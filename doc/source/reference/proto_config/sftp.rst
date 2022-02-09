.. _proto-config-sftp:

Configuration SFTP
##################

L'objet JSON de configuration du protocole SFTP est identique pour les serveurs
et les partenaires. Les options disponibles sont les suivantes :

* **keyExchanges** (*array of string*) - *Optionnel* Liste des algorithmes d'échange de clé
  autorisés sur le serveur. Les algorithmes supportés sont :

   - ``diffie-hellman-group1-sha1``
   - ``diffie-hellman-group14-sha1``
   - ``ecdh-sha2-nistp256``
   - ``ecdh-sha2-nistp384``
   - ``ecdh-sha2-nistp521``
   - ``curve25519-sha256@libssh.org``

   Par défaut, tous les algorithmes sont autorisés.

* **ciphers** (*array of string*) - *Optionnel* Liste des algorithmes de cryptage symétrique 
  de données autorisés sur le serveur. Les algorithmes supportés sont :

   - ``aes128-gcm@openssh.com``
   - ``aes128-ctr``
   - ``aes192-ctr``
   - ``aes256-ctr``
   - ``chacha20-poly1305@openssh.com``

   Par défaut, tous les algorithmes sont autorisés.

* **macs** (*array of string*) -  *Optionnel* Liste des algorithmes d'authentification de message 
  (MAC) autorisés sur le serveur. Les algorithmes supportés sont :

   - ``hmac-sha2-256-etm@openssh.com``
   - ``hmac-sha2-256``
   - ``hmac-sha1``
   - ``hmac-sha1-96``

   Par défaut, tous les algorithmes sont autorisés.

**Exemple**

.. code-block:: json

   {
     "keyExchanges": [
       "diffie-hellman-group1-sha1",
       "diffie-hellman-group14-sha1",
       "ecdh-sha2-nistp256",
       "ecdh-sha2-nistp384",
       "ecdh-sha2-nistp521",
       "curve25519-sha256@libssh.org"
     ],
     "ciphers": [
       "aes128-gcm@openssh.com",
       "aes128-ctr",
       "aes192-ctr",
       "aes256-ctr",
       "chacha20-poly1305@openssh.com"
     ],
     "macs": [
       "hmac-sha2-256-etm@openssh.com",
       "hmac-sha2-256",
       "hmac-sha1",
       "hmac-sha1-96"
     ]
   }
