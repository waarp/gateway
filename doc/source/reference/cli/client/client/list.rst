.. _reference-cli-client-client-list:

##################
Lister les clients
##################

.. program:: waarp-gateway client list

Affiche une liste de tous les clients remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway client list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum de clients autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier utilisateur renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les utilisateurs seront affichés. Les
   choix possibles sont:

   - par nom d'utilisateur (``name+``, ``name-``, ``protocol+`` & ``protocol-``)

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway client list --limit 10 --offset 5 --sort 'protocol+'
