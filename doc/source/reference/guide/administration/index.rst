Administration
==============

L'administration de la *gateway* se fait via la commande `waarp-gateway`.

L'invité de commande utilise l'interface REST de la *gateway* pour administrer
celle-ci. Par conséquent, toute commande envoyée à la *gateway* doit être
accompagnée de l'adresse de l'interface REST, ainsi que d'identifiants pour
l'authentification.

Toutes les commandes doivent donc commencer comme ceci :

.. code-block:: shell

   waarp-gateway https://user:mot_de_passe@127.0.0.1:8080 ...


Le mot de passe peut être omis, auquel cas, il sera demandé via l'invité de
commande.

Pour plus de détails sur les commandes disponibles, vous pouvez consulter
:doc:`la documentation du client<../../client/index>`, ou utiliser la commande
d'aide ``-h`` dans un terminal (en anglais).


.. warning::
   Lors du premier démarrage de la *gateway*, un unique utilisateur nommé 'admin'
   est créé avec comme mot de passe 'admin_password'. Cet utilisateur n'a vocation
   à être utilisé que pour créer d'autres utilisateurs pour administrer la *gateway*.
   Il est donc fortement recommandé de créer immédiatement un ou plusieurs
   utilisateurs, puis de supprimer le compte 'admin'.


.. toctree::
   :maxdepth: 2

   users
   rules
   servers
   local_accounts
   partners
   remote_accounts
   certificates
   transfers