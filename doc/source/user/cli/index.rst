.. _user-guide-client:

##########################################
Gestion de la Gateway en ligne de commande
##########################################


L'administration de la *gateway* se fait via la commande
:program:`waarp-gateway`.

L'invite de commande utilise l'interface REST de la Gateway pour administrer
celle-ci. Par conséquent, toute commande envoyée à la Gateway doit être
accompagnée de l'adresse de l'interface REST, ainsi que d'identifiants pour
l'authentification.

Toutes les commandes doivent donc commencer comme ceci :

.. code-block:: shell

   waarp-gateway -a http://user:mot_de_passe@127.0.0.1:8080 ...


Le mot de passe peut être omis, auquel cas, il sera demandé via l'invité de
commande.

Afin de ne pas avoir à saisir l'URL à chaque commande, il est possible de
l'exporter en variable d'environnement :

.. code-block:: sh

   export WAARP_GATEWAY_ADDRESS=http://utilisateur:mot_de_passe@localhost:8080

Pour plus de détails sur les commandes disponibles, vous pouvez consulter
:any:`la référence complète des commandes <reference-cli>`, ou utiliser
l'argument d'aide (``-h`` ou ``--help``) dans un terminal (en anglais).

.. warning::

   Lors du premier démarrage de la Gateway, un unique utilisateur nommé
   ``admin`` est créé avec comme mot de passe ``admin_password`` Cet utilisateur
   n'a vocation à être utilisé que pour créer d'autres utilisateurs pour
   administrer la *gateway*.  Il est donc fortement recommandé de créer
   immédiatement un ou plusieurs utilisateurs, puis de supprimer le compte
   'admin', ou au moins de changer son mot de passe.


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
