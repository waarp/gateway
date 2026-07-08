===========================
Lister les filewatchers
===========================

.. program:: waarp-gateway filewatcher list

Affiche une liste de tous les *filewatchers* remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway filewatcher list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum de *filewatchers* renvoyés. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier *filewatcher* renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les *filewatchers* seront affichés.
   Les choix possibles sont :

   - par nom de flux (``flow+``, ``flow-``)

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway filewatcher list --limit 10 --offset 5 --sort 'flow+'
