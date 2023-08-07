.. _reference-cli-client-remote-accounts-list:

##################################
Lister les comptes d'un partenaire
##################################

.. program:: waarp-gateway account remote list

Affiche un liste des comptes rattachés au partenaire donné.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum comptes à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier compte renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   L'attribut et l'ordre selon lesquels les comptes seront triés. Les choix
   possibles sont:

   - tri par login (``login+`` & ``login-``)



**Exemple**

.. code-block:: shell

   waarp-gateway -a http://user:password@remotehost:8080 account remote 'openssh' list -l 10 -o 5 -s 'login-'
