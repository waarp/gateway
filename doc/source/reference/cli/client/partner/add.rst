=====================
Ajouter un partenaire
=====================

.. program:: waarp-gateway partner add

.. describe:: waarp-gateway partner add

Ajoute un nouveau partenaire avec les attributs renseignés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du partenaire. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le partenaire.

.. option:: -a <ADDRESS>, --address=<ADDRESS>

   L'adresse du partenaire (au format [adresse:port]).

.. option:: -c <KEY:VAL>, --config=<KEY:VAL>

   La configuration protocolaire du partenaire. Répéter pour chaque paramètre de la
   configuration. Les options de la configuration varient en fonction du protocole
   utilisé (voir :ref:`configuration protocolaire <reference-proto-config>` pour
   plus de détails).

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 partner add -n waarp_sftp -p sftp -a waarp.org:2021 -c 'keyExchanges:["ecdh-sha2-nistp256"]'
