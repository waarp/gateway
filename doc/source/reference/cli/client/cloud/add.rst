==========================
Ajouter une instance cloud
==========================

.. program:: waarp-gateway cloud add

Ajoute une nouvelle instance cloud avec les informations données.

**Commande**

.. code-block:: shell

   waarp-gateway cloud add

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nom de l'instance cloud. Sera utilisé dans les chemins de transfert.
   Doit être unique.

.. option:: -t <TYPE>, --type=<TYPE>

   Le type de l'instance cloud. Voir la :ref:`liste des types d'instances cloud
   <reference-cloud>` pour la liste des types d'instances cloud supportés.

.. option:: -k <KEY>, --key=<KEY>

   La clé de connexion à l'instance cloud (si l'instance cloud en requiert une).

.. option:: -s <SECRET>, --secret=<SECRET>

   Le secret de connexion à l'instance cloud (si l'instance cloud en requiert un).

.. option:: -o <OPTION>, --option=<OPTION>

   Les options de connexion à l'instance cloud. Cet argument doit prendre la
   forme d'une pair *clé:valeur*. L'argument peut être répété plusieurs fois
   pour renseigner plusieurs options. Les options acceptées dépendent du type
   de l'instance cloud. Se référer à la section :ref:`cloud <reference-cloud>`
   du type concerné pour en avoir la liste.


**Exemple**

.. code-block:: shell

   waarp-gateway user add -n "aws" -t "s3" -k "abcdef" -s "123456" -o "bucket:waarp" -o "region:eu-west-3"
