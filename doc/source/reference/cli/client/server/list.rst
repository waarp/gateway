.. _reference-cli-client-servers-list:

###################
Lister les serveurs
###################

.. program:: waarp-gateway server list

Affiche les informations des serveurs remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway server list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Nombre de maximum de serveurs autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier serveur renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   L'ordre et le paramètre de tri des serveur renvoyés. Les choix possibles
   sont: tri par nom (``name+`` & ``name-``) ou par protocole (``protocol+`` &
   ``protocol-``)

.. option:: -p <PROTO>, --protocol=<PROTO>

   Filtre uniquement les serveurs utilisant le protocole renseigné avec ce
   paramètre. Ce paramètre peut être répété plusieurs fois pour filtrer
   plusieurs protocoles en même temps.

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|


**Exemple**

.. code-block:: shell

   waarp-gateway server list -l 10 -o 5 -s 'protocol+' -p 'r66' -p 'http'
