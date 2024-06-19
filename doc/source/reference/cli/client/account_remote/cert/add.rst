====================================================
[OBSOLÈTE] Ajouter un certificat à un compte distant
====================================================

.. program:: waarp-gateway account remote cert add

Attache un nouveau certificat au compte donné à partir des informations renseignées.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" cert "<LOGIN>" add

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le compte concerné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat TLS du compte, avec
   la chaîne de certification complète en format PEM.

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée (TLS ou SSH) du compte,
   en format PEM.

**Exemple**

.. code-block:: shell

   waarp-gateway account remote 'waarp_sftp' cert 'titi' add -n 'key_titi' -p './titi.key'
