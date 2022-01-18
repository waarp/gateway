===================================================
[OBSOLÈTE] Modifier un certificat de compte distant
===================================================

.. program:: waarp-gateway account remote cert update

.. describe:: waarp-gateway account remote <PARTNER> cert <LOGIN> update <CERT>

Change les attributs du certificat donné. Les noms du partenaire, du compte et du
certificat doivent être renseignés en arguments de programme. Les attributs omis
resteront inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour un compte donné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat TLS du serveur, avec
   la chaîne de certification complète en format PEM.

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée (TLS ou SSH) de l'agent,
   en format PEM.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' account remote 'waarp_sftp' cert 'titi' update 'key_titi' -n 'key_titi2' -p './titi2.key'
