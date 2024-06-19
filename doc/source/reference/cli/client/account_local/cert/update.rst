=================================================
[OBSOLÈTE] Modifier un certificat de compte local
=================================================

.. program:: waarp-gateway account local cert update

Remplace les attributs du certificats par ceux renseignés ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<PARTNER>" cert "<LOGIN>" update "<CERT>"

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour un compte donné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat TLS du client, avec
   la chaîne de certification complète en format PEM. Mutuellement exclusif avec
   l'option ``-b`` (ou ``--public_key``).

.. option:: -b <PUB_KEY>, --public_key=<PUB_KEY>

   Le chemin vers le fichier contenant la clé publique SSH du client en format
   ``authorized_key``. Mutuellement exclusif avec l'option ``-c`` (ou
   ``--certificate``).

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'serveur_sftp' cert 'tata' update 'key_tata' -n 'key_tata2' -b './tata2.pub'
