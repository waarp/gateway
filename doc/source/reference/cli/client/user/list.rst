.. _reference-cli-client-user-list:

#######################
Lister les utilisateurs
#######################

.. program:: waarp-gateway user list

Affiche une liste de tous les utilisateurs remplissant les critères ci-dessous.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum d'utilisateurs autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier utilisateur renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les utilisateurs seront affichés. Les
   choix possibles sont:

   - par nom d'utilisateur (``username+`` & ``username-``)

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' user list -l 10 -o 5 -s 'username+'
