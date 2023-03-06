.. _reference-cli-client-client-list:

##################
Lister les clients
##################

.. program:: waarp-gateway client list

Affiche une liste de tous les clients remplissant les critères ci-dessous.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum de clients autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier utilisateur renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les utilisateurs seront affichés. Les
   choix possibles sont:

   - par nom d'utilisateur (``name+``, ``name-``, ``protocol+`` & ``protocol-``)

|

**Exemple**

.. code-block:: shell

   waarp-gateway client list --limit 10 --offset 5 --sort 'protocol+'
