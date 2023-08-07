================================================
[OBSOLÈTE] Ajouter un certificat à un partenaire
================================================

.. program:: waarp-gateway partner cert add

Attache un nouveau certificat au partenaire donné à partir des informations renseignées.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le partenaire concerné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat TLS du partenaire, avec
   la chaîne de certification complète en format PEM. Mutuellement exclusif avec
   l'option ``-b`` (ou ``--public_key``).

.. option:: -b <PUB_KEY>, --public_key=<PUB_KEY>

   Le chemin vers le fichier contenant la clé publique SSH du partenaire en
   format ``authorized_key``. Mutuellement exclusif avec l'option ``-c`` (ou
   ``--certificate``).

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' partner cert 'waarp_sftp' add -n 'waarp_hostkey' -b './waarp.pub'
