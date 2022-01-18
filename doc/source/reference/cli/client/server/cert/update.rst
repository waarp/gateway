============================================
[OBSOLÈTE] Modifier un certificat de serveur
============================================

.. program:: waarp-gateway server cert update

.. describe:: waarp-gateway server cert <SERVER> update <CERT>

Change les attributs du certificat donné. Les noms du serveur et du certificat
doivent être renseignés en arguments de programme. Les attributs omis resteront
inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour un serveur donné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat TLS du serveur, avec
   la chaîne de certification complète en format PEM.

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée (TLS ou SSH) du serveur,
   en format PEM.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' server cert gw_r66 update 'cert_r66' -n 'cert_r66_new' -c './r66_2.crt' -p './r66_2.key'
