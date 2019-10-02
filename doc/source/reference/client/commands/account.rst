Gestion des comptes partenaires
===============================

.. program:: waarp-gateway

.. option:: account

   Commande de gestion des comptes partenaires. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   Un compte est toujours rattaché à un partenaire, et permet donc à la gateway
   de s'authentifier auprès de ce partenaire distant lors d'un transfert.

   .. option:: add

      Commande de création de compte.

      .. option:: --partner_id=<PARTNER_ID>

         L'identifiant numérique du partenaire auquel le nouveau compte sera
         rattaché.

      .. option:: -l <LOGIN>, --login=<LOGIN>

         Spécifie le login du nouveau compte créé. Doit être unique pour un
         partenaire donné.

      .. option:: -p <PASS>, --password=<PASS>

         Le mot de passe du nouveau compte.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 account add -l account1 -p password1 --partner_id=1

   .. option:: list

      Commande de listing de multiples comptes.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum comptes autorisés dans la réponse. Fixé à
         20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro du premier compte renvoyé.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les comptes seront triés. Les choix
         possibles sont: tri par login (`login`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: --partner_id=<PARTNER_ID>

         Filtre les comptes rattachés au partenaire renseigné avec ce paramètre.
         Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
         partenaires à la fois.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 account list -l 10 -o 5 -s login -d --partner_id=1 --partner_id=2

   .. option:: get <ACCOUNT_ID>

      Commande de consultation de compte. L'identifiant du compte souhaité doit
      être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 account get 1

   .. option:: update <ACCOUNT_ID>

      Commande de modification d'un compte existant. L'identifiant numérique du
      compte doit être renseigné en argument de programme.

      .. option:: --partner_id=<PARTNER_ID>

         Change le partenaire auquel le compte est rattaché.

      .. option:: -l <LOGIN>, --login=<LOGIN>

         Change le nom d'utilisateur du compte. Doit être unique pour un
         partenaire donné.

      .. option:: -p <PASS>, --password=<PASS>

         Change le mot de passe du compte.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 account update 1 -l account2 -p password2 --partner_id=2

   .. option:: delete <ACCOUNT_ID>

      Commande de suppression de compte. Le nom d'utilisateur du compte à
      supprimer doit être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 account delete 1