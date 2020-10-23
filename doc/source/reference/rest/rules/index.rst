##################
Gestion des règles
##################

Le point d'accès pour gérer les règles de transfert est ``/api/rules``.

Par défaut, les règles sont utilisables par tous les agents (serveurs, partenaires,
comptes) connus. Pour restreindre l'utilisation d'une règle, il suffit d'ajouter
au moins un agent à la liste blanche de la règle. Pour ajouter un agent à la liste
blanche, se référer aux chapitres :

- :any:`reference-rest-servers-authorize`
- :any:`reference-rest-servers-accounts-authorize`
- :any:`reference-rest-partners-authorize`
- :any:`reference-rest-partners-accounts-authorize`

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
