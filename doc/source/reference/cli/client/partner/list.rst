.. _reference-cli-client-partners-list:

######################
Lister les partenaires
######################

.. program:: waarp-gateway partner list

Affiche une liste des partenaires remplissant les critères renseignés ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway partner list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum de partenaires affichés. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier partenaire renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   L'ordre et le paramètre de tri des partenaire affichés. Les choix
   possibles sont: tri par nom (``name+`` & ``name-``) ou par protocole
   (``protocol+`` & ``protocol-``)

.. option:: -p <PROTO>, --protocol=<PROTO>

   Filtre uniquement les partenaires utilisant le protocole renseigné avec ce
   paramètre. Le paramètre peut être répété plusieurs fois pour filtrer
   plusieurs protocoles en même temps.

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway partner list -l 10 -o 5 -s 'protocol+' -p 'sftp' -p 'http'
