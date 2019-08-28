**********************
Lister les certificats
**********************

.. http:get:: /api/certificates

   Renvoie une liste des certificats emplissant les critères données en paramètre
   de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les certificats seront triés *(défaut: name)*
   :type sortby: [name]
   :param order: L'ordre dans lequel les certificats sont triés *(défaut: asc)*
   :type order: [asc|desc]
   :param account: Filtre uniquement les certificats rattaché au compte portant ce numéro.
                   Peut être renseigné plusieurs fois pour filtrer plusieurs comptes.
   :type account: uint64

   **Exemple de requête**

       .. code-block:: http

          GET /api/certificates?limit=5 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array Certificates: La liste des certificats demandés
   :resjsonarr number ID: Le numéro unique du certificat
   :resjsonarr string Name: Le nom du certificat
   :resjsonarr number AccountID: Le numéro du compte auquel appartient le certificat
   :resjsonarr string PrivateKey: La clé privée du compte
   :resjsonarr string PublicKey: La clé publique du compte
   :resjsonarr string Cert: Le certificat de la clé publique

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 453

          {
            "Certificates": [{
              "ID": "1234",
              "Name": "certificat1",
              "PartnerID": "12345",
              "PrivateKey": "*clé privée*",
              "PublicKey": "*clé publique*",
              "PrivateCert": "*certificat privée*",
              "PublicCert": "*certificat public*"
            },{
              "ID": "5678",
              "Name": "certificat2",
              "PartnerID": "67890",
              "PrivateKey": "*clé privée*",
              "PublicKey": "*clé publique*",
              "PrivateCert": "*certificat privée*",
              "PublicCert": "*certificat public*"
            }]
          }