##################
Gestion des règles
##################

Le point d'accès pour gérer les règles de transfert est ``/api/rules``.

Par défaut, les règles sont utilisables par tous les agents (serveurs, partenaires,
comptes) connus. Pour restreindre l'utilisation d'une règle, il suffit d'ajouter
au moins un agent à la liste blanche de la règle. Pour ajouter un agent à la liste
blanche, se référer aux chapitres :

- :doc:`../servers/authorize`
- :doc:`../servers/accounts/authorize`
- :doc:`../partners/authorize`
- :doc:`../partners/accounts/authorize`

**Sommaire**

.. toctree::
   :maxdepth: 2

   create
   list
   consult
   update
   replace
   delete
   allow_all