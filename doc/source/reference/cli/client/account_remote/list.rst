.. _reference-cli-client-remote-accounts-list:

##################################
Lister les comptes d'un partenaire
##################################

.. program:: waarp-gateway account remote list

Affiche un liste des comptes rattachés au partenaire donné.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum comptes à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier compte renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   L'attribut et l'ordre selon lesquels les comptes seront triés. Les choix
   possibles sont:

   - tri par login (``login+`` & ``login-``)

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway account remote 'openssh' list -l 10 -o 5 -s 'login-'
