=============================================
[OBSOLÈTE] Ajouter un certificat à un serveur
=============================================

.. program:: waarp-gateway server cert add

**Commande**

.. code-block:: shell

   waarp-gateway server cert "<SERVER>" add

**Options**

Attache un nouveau certificat au serveur donné à partir des informations renseignées.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le serveur concerné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat TLS du serveur, avec
   la chaîne de certification complète en format PEM.

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée (TLS ou SSH) du serveur,
   en format PEM.

**Exemple**

.. code-block:: shell

   waarp-gateway server cert 'gw_r66' add -n 'cert_r66' -c '/r66.crt' -p '/r66.key'
