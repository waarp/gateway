=====================
Ajouter un partenaire
=====================

.. program:: waarp-gateway partner add

.. describe:: waarp-gateway <ADDR> partner add

Ajoute un nouveau partenaire avec les attributs renseignés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du partenaire. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le partenaire.

.. option:: -c <CONF>, --config=<CONF>

   La configuration protocolaire du partenaire en format JSON. Contient les
   informations nécessaires pour se connecter au partenaire. Le contenu de la
   configuration varie en fonction du protocole utilisé.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 partner add -n waarp_sftp -p sftp -c '{"address":"waarp.org","port":21}'
