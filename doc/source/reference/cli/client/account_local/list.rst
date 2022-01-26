.. _reference-cli-client-local-accounts-list:

###############################
Lister les comptes d'un serveur
###############################

.. program:: waarp-gateway account local list

.. describe:: waarp-gateway account local <SERVER> list

Affiche une liste des comptes remplissant les critères donnés ci-dessous.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum comptes à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier compte affiché.

.. option:: -s <SORT>, --sort=<SORT>

   L'attribut et l'ordre selon lesquels les comptes seront triés. Les choix
   possibles sont:

   - tri par login (``login+`` & ``login-``)

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a http://user:password@localhost:8080 account local serveur_sftp list -l 10 -o 5 -s login-
