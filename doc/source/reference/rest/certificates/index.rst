#######################
Gestion des certificats
#######################

Le point d'accès pour gérer les certificats est ``api/certificates``.

Un certificat est toujours rattaché à une entité pouvant être:

- un serveur local (``local_agents``)
- un partenaire distant (``remote_agents``)
- un compte local (``local_accounts``)
- un compte partenaire (``remote_accounts``).

L'entité en question est définie dans l'objet ``certificate`` par la combinaison
du nom de sa table (``ownerType``) et de son identifiant dans cette table
(``ownerID``).

.. toctree::
   :maxdepth: 1

   create
   list
   consult
   update
   replace
   delete