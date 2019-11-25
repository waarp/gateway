Gestion des règles de transfert
===============================

.. program:: waarp-gateway

.. option:: rule

   Commande de gestion des règles de transfert. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   .. option:: add

      Commande de création de règle.

      .. option:: -n <NAME>, --name=<NAME>

         Le nom de la règle de transfert. Doit être unique.

      .. option:: -c <COMMENT>, --comment=<COMMENT>

         Un commentaire optionnel décrivant la règle.

      .. option:: -d <DIRECTION>, --name=<DIRECTION>

         Spécifie le sens dans lequel la règle peut être utilisée. Une règle
         peut être utilisée en réception (*RECEIVE*) ou en envoi (*SEND*).

      .. option:: -p <PATH>, --path=<PATH>

         Le chemin associé à la règle. Détermine le dossier de destination des
         transferts utilisant cette règle. Si le transfert est en réception, il
         s'agit d'un chemin local. Si le transfert est en envoi, il s'agit d'un
         chemin sur le partenaire distant. Le chemin doit être unique.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 rule add -n "réception sftp" -c "règle de réception des fichiers avec SFTP" -d RECEIVE -p "/sftp/reception"

   .. option:: get <RULE_ID>

      Commande de consultation de règle. L'identifiant de la règle à renvoyer
      doit être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 rule get 1

   .. option:: list

      Commande de listing de multiples règles.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum règles autorisées dans la réponse. Fixé à
         20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro de la première règle renvoyée.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les règles seront triés. Les choix
         possibles sont: tri par nom (`name`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 rule list 1 -l 10 -o 5 -s name -d

   .. option:: delete <RULE_ID>

      Commande de suppression de règle. L'identifiant de la règle à supprimer
      doit être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 rule delete 1

   .. option:: access

      Commande de gestion de l'accès aux règles. Seules les entités (serveurs ou
      compte) ayant accès à une règle peuvent l'utiliser. Une règle n'ayant pas
      d'accès est considérée comme utilisable par toutes les entités connues.

      .. option:: grant <RULE_ID>

         Autorise l'accès à la règle portant l'identifiant *RULE_ID*, qui doit
         être spécifié en argument de programme.

         .. option:: -t <TYPE>, --type=<TYPE>

            Le type d'entitée à laquelle l'accès à la règle est accordé. Peut
            être un serveur local (*local agent*), un serveur distant
            (*remote agent*), un compte local (*local account*) ou un compte
            distant (*remote account*).

         .. option:: -i <ID>, --id=<ID>

            L'identifiant de l'entitée à laquelle l'accès est accordé.

         **Exemple**

         .. code-block:: bash

            waarp-gateway -a http://user:password@localhost:8080 rule access grant -t "local agent" -i 1 1

      .. option:: revoke <RULE_ID>

         Révoque l'accès à la règle portant l'identifiant *RULE_ID*, qui doit
         être spécifié en argument de programme.

         .. option:: -t <TYPE>, --type=<TYPE>

            Le type d'entitée à laquelle l'accès à la règle est révoqué. Peut
            être un serveur local (*local agent*), un serveur distant
            (*remote agent*), un compte local (*local account*) ou un compte
            distant (*remote account*).

         .. option:: -i <ID>, --id=<ID>

            L'identifiant de l'entitée à laquelle l'accès est révoqué.

         .. code-block:: bash

            waarp-gateway -a http://user:password@localhost:8080 rule access revoke -t "local agent" -i 1 1

      .. option:: list <RULE_ID>

         Liste tous les accès accordé pour la règle portant l'identifiant
         *RULE_ID*, qui doit être spécifié en argument de programme.

         **Exemple**

         .. code-block:: bash

            waarp-gateway -a http://user:password@localhost:8080 rule access list 1

   .. option:: tasks

      Commande de gestion des chaînes de traitements d'une règle.

      .. option:: change <RULE_ID>

         Change les chaînes de traitement de la règle portant l'identifiant
         *RULE_ID*. Attention, si une des chaînes est laissée vide, toutes les
         tâches de cette chaîne seront supprimée et le chaîne sera considérée
         comme vide.

         Les chaînes doivent être renseignées sous la forme de tableaux JSON
         (voir :doc:`la documentation REST <../../rest/rules/tasks/update>` pour
         plus d'information sur la structure du JSON).

         .. option:: --pre=<PRE_TASKS>

            La liste des pré-traitements de la règle. Ces traitements seront
            lancés, dans l'ordre, avant le transfert du fichier.

         .. option:: --ost=<POST_TASKS>

            La liste des post-traitements de la règle. Ces traitements seront
            lancés, dans l'ordre, après le transfert du fichier.

         .. option:: --error=<ERROR_TASKS>

            La liste des traitements d'erreur de la règles. Ces traitements
            seront lancés, dans l'ordre, si une erreur se produit pendant le
            transfert ou pendant les pré/post-traitements.

         **Exemple**

         .. code-block:: bash

            waarp-gateway -a http://user:password@localhost:8080 rule tasks change --pre='[{"type":"COPY", "args":"{\"dst\":\"copy/destination\"}"}]' --post='[{"type":"DELETE","args":"{}"}]' --error='[{"type":"EXEC","args":"{\"target\":\"program\"}"}]'